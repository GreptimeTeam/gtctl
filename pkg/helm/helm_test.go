package helm

import (
	"strings"
	"testing"
)

func TestGetUrlFromRemoteIndex(t *testing.T) {
	var getUrlTests = []struct {
		input  string
		output string
	}{
		{
			"greptimedb-operator",
			"https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-operator",
		},
		{
			"greptimedb",
			"https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb",
		},
		{
			"greptimedb-etcd",
			"https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-etcd",
		},
	}

	for _, v := range getUrlTests {
		index, err := GetLatestChart()
		if err != nil {
			t.Errorf("Get latest chart error:%v", err)
			return
		}
		url := index.Entries[v.input][0].URLs[0]
		if strings.Contains(url, v.output) {
			t.Errorf("Download url invalid, input:%s, download url:%s", v.input, url)
		}
	}
}
