// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
)

const (
	helmFieldTag = "helm"
)

var (
	// KubeVersion is the target version of the kubernetes.
	KubeVersion = "v1.20.0"
)

// Manager is the Helm charts manager. The implementation is based on Helm SDK.
// The main purpose of Manager is:
// 1. Load the chart from remote charts and save them in cache directory.
// 2. Generate the manifests from the chart with the values.
type Manager struct {
	// logger is the logger for the Manager.
	logger logger.Logger

	// am is the artifacts manager to manage charts.
	am artifacts.Manager

	// mm is the metadata manager to manage the metadata.
	mm metadata.Manager
}

type Option func(*Manager)

func NewManager(l logger.Logger, opts ...Option) (*Manager, error) {
	r := &Manager{logger: l}

	am, err := artifacts.NewManager(l)
	if err != nil {
		return nil, err
	}
	r.am = am

	mm, err := metadata.New("")
	if err != nil {
		return nil, err
	}
	r.mm = mm

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

func WithHomeDir(dir string) Option {
	return func(r *Manager) {
		metadataManager, err := metadata.New(dir)
		if err != nil {
			r.logger.Errorf("failed to create metadata manager: %v", err)
			os.Exit(1)
		}
		r.mm = metadataManager
	}
}

// LoadAndRenderChart loads the chart from the remote charts and render the manifests with the values.
func (r *Manager) LoadAndRenderChart(ctx context.Context, name, namespace, chartName, chartVersion string, useGreptimeCNArtifacts bool, options interface{}) ([]byte, error) {
	values, err := r.generateHelmValues(options)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with values: %v", name, values)

	if chartVersion == "" {
		chartVersion = artifacts.LatestVersionTag
	}

	src, err := r.am.NewSource(chartName, chartVersion, artifacts.ArtifactTypeChart, useGreptimeCNArtifacts)
	if err != nil {
		return nil, err
	}

	destDir, err := r.mm.AllocateArtifactFilePath(src, false)
	if err != nil {
		return nil, err
	}

	chartFile, err := r.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{UseCache: true})
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(chartFile)
	if err != nil {
		return nil, err
	}
	helmChart, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	manifests, err := r.generateManifests(ctx, name, namespace, helmChart, values)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with manifests: %s", name, string(manifests))

	return manifests, nil
}

func (r *Manager) generateManifests(ctx context.Context, releaseName, namespace string,
	chart *chart.Chart, values map[string]interface{}) ([]byte, error) {
	client, err := r.newHelmClient(releaseName, namespace)
	if err != nil {
		return nil, err
	}

	rel, err := client.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, err
	}

	var manifests bytes.Buffer
	_, err = fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	if err != nil {
		return nil, err
	}

	return manifests.Bytes(), nil
}

func (r *Manager) generateHelmValues(input interface{}) (map[string]interface{}, error) {
	var rawArgs []string
	valueOf := reflect.ValueOf(input)

	// Make sure we are handling with a struct here.
	if valueOf.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid input type, should be struct")
	}

	typeOf := reflect.TypeOf(input)
	for i := 0; i < valueOf.NumField(); i++ {
		helmValueKey := typeOf.Field(i).Tag.Get(helmFieldTag)
		if len(helmValueKey) > 0 && valueOf.Field(i).Len() > 0 {
			if helmValueKey == "*" {
				rawArgs = append(rawArgs, valueOf.Field(i).String())
			} else {
				rawArgs = append(rawArgs, fmt.Sprintf("%s=%s", helmValueKey, valueOf.Field(i)))
			}
		}
	}

	if len(rawArgs) > 0 {
		values := make(map[string]interface{})
		if err := strvals.ParseInto(strings.Join(rawArgs, ","), values); err != nil {
			return nil, err
		}
		return values, nil
	}

	return nil, nil
}

func (r *Manager) newHelmClient(releaseName, namespace string) (*action.Install, error) {
	kubeVersion, err := chartutil.ParseKubeVersion(KubeVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid kube version '%s': %s", kubeVersion, err)
	}

	helmClient := action.NewInstall(new(action.Configuration))
	helmClient.DryRun = true
	helmClient.ReleaseName = releaseName
	helmClient.Replace = true
	helmClient.ClientOnly = true
	helmClient.IncludeCRDs = true
	helmClient.Namespace = namespace
	helmClient.KubeVersion = kubeVersion

	return helmClient, nil
}
