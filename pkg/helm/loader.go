// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
)

var (
	// KubeVersion is the target version of the kubernetes.
	KubeVersion = "v1.20.0"
)

// Loader is the Helm charts loader. The implementation is based on Helm SDK.
// The main purpose of Loader is:
// 1. Load the chart from remote charts and save them in cache directory.
// 2. Generate the manifests from the chart with the values.
type Loader struct {
	// logger is the logger for the Loader.
	logger logger.Logger

	// am is the artifacts manager to manage charts.
	am artifacts.Manager

	// mm is the metadata manager to manage the metadata.
	mm metadata.Manager
}

type Option func(*Loader)

func NewLoader(l logger.Logger, opts ...Option) (*Loader, error) {
	r := &Loader{logger: l}

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
	return func(r *Loader) {
		mm, err := metadata.New(dir)
		if err != nil {
			r.logger.Errorf("failed to create metadata manager: %v", err)
			os.Exit(1)
		}
		r.mm = mm
	}
}

// LoadOptions is the options for running LoadAndRenderChart.
type LoadOptions struct {
	// ReleaseName is the name of the release.
	ReleaseName string

	// Namespace is the namespace of the release.
	Namespace string

	// ChartName is the name of the chart.
	ChartName string

	// ChartVersion is the version of the chart.
	ChartVersion string

	// FromCNRegion indicates whether to use the artifacts from CN region.
	FromCNRegion bool

	// ValuesOptions is the options for generating the helm values.
	ValuesOptions interface{}

	// ValuesFile is the path to the values file.
	ValuesFile string

	// EnableCache indicates whether to enable the cache.
	EnableCache bool
}

// LoadAndRenderChart loads the chart from the remote charts and render the manifests with the values.
func (r *Loader) LoadAndRenderChart(ctx context.Context, opts *LoadOptions) ([]byte, error) {
	values, err := ToHelmValues(opts.ValuesOptions, opts.ValuesFile)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with values: %v", opts.ReleaseName, values)

	if opts.ChartVersion == "" {
		opts.ChartVersion = artifacts.LatestVersionTag
	}

	src, err := r.am.NewSource(opts.ChartName, opts.ChartVersion, artifacts.ArtifactTypeChart, opts.FromCNRegion)
	if err != nil {
		return nil, err
	}

	destDir, err := r.mm.AllocateArtifactFilePath(src, false)
	if err != nil {
		return nil, err
	}

	chartFile, err := r.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{EnableCache: opts.EnableCache})
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

	manifests, err := r.generateManifests(ctx, opts.ReleaseName, opts.Namespace, helmChart, values)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with manifests: %s", opts.ReleaseName, string(manifests))

	return manifests, nil
}

func (r *Loader) generateManifests(ctx context.Context, releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error) {
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

func (r *Loader) newHelmClient(releaseName, namespace string) (*action.Install, error) {
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
