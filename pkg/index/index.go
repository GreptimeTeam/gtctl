package index

import (
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/blang/semver/v4"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	GreptimeHelmChartsIndex = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"
)

func GetUrlFromRemoteIndex(chartName string) (string, error) {
	rsp, err := http.Get(GreptimeHelmChartsIndex)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}

	var index repo.IndexFile
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return "", err
	}

	sort.Slice(index.Entries[chartName], func(i, j int) bool {
		in := index.Entries[chartName]
		versionA, _ := semver.Parse(in[i].Version)
		versionB, _ := semver.Parse(in[j].Version)
		return versionB.LT(versionA)
	})

	return index.Entries[chartName][0].URLs[0], nil
}
