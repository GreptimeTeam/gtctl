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
	"reflect"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	. "helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"
)

const (
	helmFieldTag = "helm"
)

type TemplateRender interface {
	GenerateManifests(releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error)
}

type Render struct {
	// indexFile is the index file of the remote charts.
	indexFile *IndexFile
}

var _ TemplateRender = &Render{}

func (r *Render) LoadChartFromRemoteCharts(downloadURL string) (*chart.Chart, error) {
	rsp, err := http.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return loader.LoadArchive(bytes.NewReader(body))
}

func (r *Render) LoadChartFromLocalDirectory(directory string) (*chart.Chart, error) {
	return loader.LoadDir(directory)
}

func (r *Render) GenerateManifests(releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error) {
	client, err := r.newHelmClient(releaseName, namespace)
	if err != nil {
		return nil, err
	}

	rel, err := client.RunWithContext(context.TODO(), chart, values)
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
		if helmValueKey != "" && valueOf.Field(i).Len() > 0 {
			rawArgs = append(rawArgs, fmt.Sprintf("%s=%s\n", helmValueKey, valueOf.Field(i)))
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

func (r *Render) GetIndexFile(indexURL string) (*IndexFile, error) {
	if r.indexFile != nil {
		return r.indexFile, nil
	}

	rsp, err := http.Get(indexURL)
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
