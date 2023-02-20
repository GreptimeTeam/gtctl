// Copyright 2022 Greptime Team
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
	"net/http"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type TemplateRender interface {
	GenerateManifests(releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error)
}

type Render struct{}

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
