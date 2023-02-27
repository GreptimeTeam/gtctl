package helm

import (
	"sort"
	"testing"

	"github.com/Masterminds/semver"
)

const (
	testChartName = "greptimedb"
)

func TestRender_GetIndexFile(t *testing.T) {
	tests := []struct {
		url string
	}{
		{
			url: "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml",
		},
		{
			url: "https://github.com/kubernetes/kube-state-metrics/raw/gh-pages/index.yaml",
		},
	}

	r := &Render{}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, err := r.GetIndexFile(tt.url)
			if err != nil {
				t.Errorf("fetch index '%s' failed, err: %v", tt.url, err)
			}
		})
	}
}

func TestRender_GetLatestChartLatestChart(t *testing.T) {
	tests := []struct {
		url string
	}{
		{
			url: "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml",
		},
	}
	r := &Render{}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			indexFile, err := r.GetIndexFile(tt.url)
			if err != nil {
				t.Errorf("fetch index '%s' failed, err: %v", tt.url, err)
			}

			chart, err := r.GetLatestChart(indexFile, testChartName)
			if err != nil {
				t.Errorf("get latest chart failed, err: %v", err)
			}

			var rawVersions []string
			for _, v := range indexFile.Entries[testChartName] {
				rawVersions = append(rawVersions, v.Version)
			}

			vs := make([]*semver.Version, len(rawVersions))
			for i, r := range rawVersions {
				v, err := semver.NewVersion(r)
				if err != nil {
					t.Errorf("Error parsing version: %s", err)
				}

				vs[i] = v
			}

			sort.Sort(semver.Collection(vs))

			if chart.Version != vs[len(vs)-1].String() {
				t.Errorf("latest chart version not match, expect: %s, got: %s", vs[len(vs)-1].String(), chart.Version)
			}
		})
	}
}
