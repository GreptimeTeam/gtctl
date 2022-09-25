package helm

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/GreptimeTeam/gtctl/charts"
)

type TemplateRender interface {
	GenerateManifests(releaseName, namespace string, chart *chart.Chart, values map[string]interface{}) ([]byte, error)
}

type Render struct{}

var _ TemplateRender = &Render{}

// TODO(zyy17): Support remote charts.

func (r *Render) LoadChartFromLocalDirectory(directory string) (*chart.Chart, error) {
	return loader.LoadDir(directory)
}

func (r *Render) LoadChartFromEmbedCharts(application, version string) (*chart.Chart, error) {
	chartName := fmt.Sprintf("%s-%s.tgz", application, version)
	content, err := charts.Charts.ReadFile(chartName)
	if err != nil {
		return nil, fmt.Errorf("chart '%s' not found: %s", chartName, err)
	}
	return loader.LoadArchive(bytes.NewReader(content))
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
