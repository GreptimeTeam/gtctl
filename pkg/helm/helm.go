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
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	. "helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"

	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

const (
	helmFieldTag = "helm"
)

const (
	defaultChartsCache = ".gtctl/charts-cache"
)

type TemplateRender interface {
	GenerateManifests(ctx context.Context, releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error)
}

type Render struct {
	// indexFile is the index file of the remote charts.
	indexFile *IndexFile

	// chartCache is the cache directory for the charts.
	chartsCacheDir string
}

var _ TemplateRender = &Render{}

func NewRender() (*Render, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	chartsCacheDir := filepath.Join(homeDir, defaultChartsCache)

	if err := fileutils.CreateDirIfNotExists(chartsCacheDir); err != nil {
		return nil, err
	}

	return &Render{
		chartsCacheDir: chartsCacheDir,
	}, nil
}

func (r *Render) LoadChartFromRemoteCharts(ctx context.Context, downloadURL string) (*chart.Chart, error) {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}

	var (
		packageName = path.Base(parsedURL.Path)
		cachePath   = filepath.Join(r.chartsCacheDir, packageName)
	)

	if r.isInChartsCache(packageName) {
		data, err := os.ReadFile(cachePath)
		if err != nil {
			return nil, err
		}
		return loader.LoadArchive(bytes.NewReader(data))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(cachePath, body, 0644); err != nil {
		return nil, err
	}

	return loader.LoadArchive(bytes.NewReader(body))
}

func (r *Render) LoadChartFromLocalDirectory(directory string) (*chart.Chart, error) {
	return loader.LoadDir(directory)
}

func (r *Render) GenerateManifests(ctx context.Context, releaseName, namespace string,
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

func (r *Render) GenerateHelmValues(input interface{}) (map[string]interface{}, error) {
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

func (r *Render) GetLatestChart(indexFile *IndexFile, chartName string) (*ChartVersion, error) {
	if versions, ok := indexFile.Entries[chartName]; ok {
		if versions.Len() > 0 {
			// The Entries are already sorted by version so the position 0 always point to the latest version.
			v := []*ChartVersion(versions)
			if len(v[0].URLs) == 0 {
				return nil, fmt.Errorf("no download URLs found for %s-%s", chartName, v[0].Version)
			}
			return v[0], nil
		}
		return nil, fmt.Errorf("chart %s has empty versions", chartName)
	}

	return nil, fmt.Errorf("chart %s not found", chartName)
}

func (r *Render) GetIndexFile(ctx context.Context, indexURL string) (*IndexFile, error) {
	if r.indexFile != nil {
		return r.indexFile, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	indexFile, err := loadIndex(body, indexURL)
	if err != nil {
		return nil, err
	}

	// Cache the index file, so we don't request the index every time.
	r.indexFile = indexFile

	return indexFile, nil
}

// Pull pulls the chart from the remote OCI registry, for example, oci://registry-1.docker.io/bitnamicharts/etcd.
func (r *Render) Pull(_ context.Context, OCIRegistry, version string) (*chart.Chart, error) {
	packageName := r.packageName(path.Base(OCIRegistry), version)
	if !r.isInChartsCache(packageName) {
		registryClient, err := registry.NewClient(
			registry.ClientOptDebug(false),
			registry.ClientOptEnableCache(false),
			registry.ClientOptCredentialsFile(""),
		)
		if err != nil {
			return nil, err
		}

		cfg := new(action.Configuration)
		cfg.RegistryClient = registryClient

		// Create a pull action
		client := action.NewPullWithOpts(action.WithConfig(cfg))
		client.Settings = cli.New()
		client.Version = version
		client.DestDir = r.chartsCacheDir

		// Execute the pull action
		if _, err := client.Run(OCIRegistry); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(filepath.Join(r.chartsCacheDir, packageName))
	if err != nil {
		return nil, err
	}

	return loader.LoadArchive(bytes.NewReader(data))
}

func (r *Render) isInChartsCache(packageName string) bool {
	res, _ := fileutils.IsFileExists(filepath.Join(r.chartsCacheDir, packageName))
	return res
}

func (r *Render) newHelmClient(releaseName, namespace string) (*action.Install, error) {
	helmClient := action.NewInstall(new(action.Configuration))
	helmClient.DryRun = true
	helmClient.ReleaseName = releaseName
	helmClient.Replace = true
	helmClient.ClientOnly = true
	helmClient.IncludeCRDs = true
	helmClient.Namespace = namespace

	return helmClient, nil
}

func (r *Render) packageName(chartName, version string) string {
	return fmt.Sprintf("%s-%s.tgz", chartName, version)
}

// loadIndex is from 'helm/pkg/index.go'.
func loadIndex(data []byte, source string) (*IndexFile, error) {
	i := &IndexFile{}

	if len(data) == 0 {
		return i, ErrEmptyIndexYaml
	}

	if err := yaml.UnmarshalStrict(data, i); err != nil {
		return i, err
	}

	for name, cvs := range i.Entries {
		for idx := len(cvs) - 1; idx >= 0; idx-- {
			if cvs[idx] == nil {
				log.Printf("skipping loading invalid entry for chart %q from %s: empty entry", name, source)
				continue
			}
			if cvs[idx].APIVersion == "" {
				cvs[idx].APIVersion = chart.APIVersionV1
			}
			if err := cvs[idx].Validate(); err != nil {
				log.Printf("skipping loading invalid entry for chart %q %q from %s: %s", name, cvs[idx].Version, source, err)
				cvs = append(cvs[:idx], cvs[idx+1:]...)
			}
		}
	}
	i.SortEntries()
	if i.APIVersion == "" {
		return i, ErrNoAPIVersion
	}
	return i, nil
}
