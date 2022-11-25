package helm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/GreptimeTeam/gtctl/third_party/index"
)

const (
	GreptimeHelmChartsIndex = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"
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

	body, err := ioutil.ReadAll(rsp.Body)
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

func GetLatestChart() (*index.IndexFile, error) {
	rsp, err := http.Get(GreptimeHelmChartsIndex)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	var index *index.IndexFile
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return nil, err
	}

	sort.Sort(index.Entries["greptimedb-operator"])
	sort.Sort(index.Entries["greptimedb"])
	sort.Sort(index.Entries["greptimedb-etcd"])

	return index, nil
}
