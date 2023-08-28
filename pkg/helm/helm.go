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

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

const (
	helmFieldTag = "helm"
)

// Manager is the Helm charts manager. The implementation is based on Helm SDK.
// The main purpose of Manager is:
// 1. Load the chart from remote charts and save them in cache directory.
// 2. Generate the manifests from the chart with the values.
type Manager struct {
	// indexFile is the index file of the remote charts.
	indexFile *IndexFile

	// chartCache is the cache directory for the charts.
	chartsCacheDir string

	// logger is the logger for the Manager.
	logger logger.Logger
}

type Option func(*Manager)

func NewManager(l logger.Logger, opts ...Option) (*Manager, error) {
	r := &Manager{logger: l}
	for _, opt := range opts {
		opt(r)
	}

	if r.chartsCacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		r.chartsCacheDir = filepath.Join(homeDir, DefaultChartsCache)
	}

	if err := fileutils.CreateDirIfNotExists(r.chartsCacheDir); err != nil {
		return nil, err
	}

	return r, nil
}

func WithChartsCacheDir(chartsCacheDir string) func(*Manager) {
	return func(r *Manager) {
		r.chartsCacheDir = chartsCacheDir
	}
}

// LoadAndRenderChart loads the chart from the remote charts and render the manifests with the values.
func (r *Manager) LoadAndRenderChart(ctx context.Context, name, namespace, chartName, chartVersion string, useGreptimeCNArtifacts bool, options interface{}) ([]byte, error) {
	values, err := r.generateHelmValues(options)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with values: %v", name, values)

	var helmChart *chart.Chart
	if isOCIChar(chartName) {
		helmChart, err = r.pullFromOCIRegistry(chartName, chartVersion)
		if err != nil {
			return nil, err
		}
	} else {
		downloadURL, err := r.getChartDownloadURL(ctx, chartName, chartVersion, useGreptimeCNArtifacts)
		if err != nil {
			return nil, err
		}

		alwaysDownload := false
		if chartVersion == "" { // always download the latest version.
			alwaysDownload = true
		}

		helmChart, err = r.loadChartFromRemoteCharts(ctx, downloadURL, alwaysDownload)
		if err != nil {
			return nil, err
		}
	}

	manifests, err := r.generateManifests(ctx, name, namespace, helmChart, values)
	if err != nil {
		return nil, err
	}
	r.logger.V(3).Infof("create '%s' with manifests: %s", name, string(manifests))

	return manifests, nil
}

func (r *Manager) loadChartFromRemoteCharts(ctx context.Context, downloadURL string, alwaysDownload bool) (*chart.Chart, error) {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}

	var (
		packageName = path.Base(parsedURL.Path)
		cachePath   = filepath.Join(r.chartsCacheDir, packageName)
	)

	if !alwaysDownload && r.isInChartsCache(packageName) {
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

func (r *Manager) getLatestChart(indexFile *IndexFile, chartName string) (*ChartVersion, error) {
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

func (r *Manager) getIndexFile(ctx context.Context, indexURL string) (*IndexFile, error) {
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

// pullFromOCIRegistry pulls the chart from the remote OCI registry, for example, oci://registry-1.docker.io/bitnamicharts/etcd.
func (r *Manager) pullFromOCIRegistry(chartsRegistry, version string) (*chart.Chart, error) {
	packageName := r.packageName(path.Base(chartsRegistry), version)
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

		r.logger.V(3).Infof("pulling chart '%s', version: '%s' from OCI registry", chartsRegistry, version)
		// Execute the pull action
		if _, err := client.Run(chartsRegistry); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(filepath.Join(r.chartsCacheDir, packageName))
	if err != nil {
		return nil, err
	}

	return loader.LoadArchive(bytes.NewReader(data))
}

func (r *Manager) isInChartsCache(packageName string) bool {
	res, _ := fileutils.IsFileExists(filepath.Join(r.chartsCacheDir, packageName))
	if res {
		r.logger.V(3).Infof("chart '%s' is already in cache", packageName)
	}
	return res
}

func (r *Manager) newHelmClient(releaseName, namespace string) (*action.Install, error) {
	helmClient := action.NewInstall(new(action.Configuration))
	helmClient.DryRun = true
	helmClient.ReleaseName = releaseName
	helmClient.Replace = true
	helmClient.ClientOnly = true
	helmClient.IncludeCRDs = true
	helmClient.Namespace = namespace

	return helmClient, nil
}

func (r *Manager) getChartDownloadURL(ctx context.Context, chartName, version string, useGreptimeCNArtifacts bool) (string, error) {
	// Get the latest version from index file of GitHub repo.
	if !useGreptimeCNArtifacts && version == "" {
		indexFile, err := r.getIndexFile(ctx, GreptimeChartIndexURL)
		if err != nil {
			return "", err
		}

		chartVersion, err := r.getLatestChart(indexFile, chartName)
		if err != nil {
			return "", err
		}

		downloadURL := chartVersion.URLs[0]
		r.logger.V(3).Infof("get latest chart '%s', version '%s', url: '%s'",
			chartName, chartVersion.Version, downloadURL)
		return downloadURL, nil
	}

	if useGreptimeCNArtifacts {
		if version == "" {
			version = "latest"
		}

		// The download URL example: 'https://downloads.greptime.cn/releases/charts/etcd/9.2.0/etcd-9.2.0.tgz'.
		downloadURL := fmt.Sprintf("%s/%s/%s/%s.tgz", GreptimeCNCharts, chartName, version, chartName+"-"+version)
		r.logger.V(3).Infof("get given version chart '%s', version '%s', url: '%s'",
			chartName, version, downloadURL)
		return downloadURL, nil
	}

	// The download URL example: 'https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-0.1.1-alpha.3/greptimedb-0.1.1-alpha.3.tgz'.
	downloadURL := fmt.Sprintf("%s/%s/%s.tgz", GreptimeChartReleaseDownloadURL, chartName, chartName+"-"+version)
	r.logger.V(3).Infof("get given version chart '%s', version '%s', url: '%s'",
		chartName, version, downloadURL)

	return downloadURL, nil
}

func (r *Manager) packageName(chartName, version string) string {
	return fmt.Sprintf("%s-%s.tgz", chartName, version)
}

func isOCIChar(url string) bool {
	return strings.HasPrefix(url, "oci://")
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
