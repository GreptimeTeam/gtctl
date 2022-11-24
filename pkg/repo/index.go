package repo

import (
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/blang/semver/v4"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

const (
	defaultIndexURL = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"
)

type ChartVersions []*ChartVersion

type IndexFile struct {
	// This is used ONLY for validation against chartmuseum's index files and is discarded after validation.
	ServerInfo map[string]interface{}   `yaml:"serverInfo,omitempty"`
	APIVersion string                   `yaml:"apiVersion"`
	Generated  time.Time                `yaml:"generated"`
	Entries    map[string]ChartVersions `yaml:"entries"`
	PublicKeys []string                 `yaml:"publicKeys,omitempty"`

	// Annotations are additional mappings uninterpreted by Helm. They are made available for
	// other applications to add information to the index file.
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type ChartVersion struct {
	Meta    *chart.Metadata
	URLs    []string  `yaml:"urls"`
	Created time.Time `yaml:"created,omitempty"`
	Removed bool      `yaml:"removed,omitempty"`
	Digest  string    `yaml:"digest,omitempty"`
	Version string    `yaml:"version,omitempty"`

	// ChecksumDeprecated is deprecated in Helm 3, and therefore ignored. Helm 3 replaced
	// this with Digest. However, with a strict YAML parser enabled, a field must be
	// present on the struct for backwards compatibility.
	ChecksumDeprecated string `yaml:"checksum,omitempty"`

	// EngineDeprecated is deprecated in Helm 3, and therefore ignored. However, with a strict
	// YAML parser enabled, this field must be present.
	EngineDeprecated string `yaml:"engine,omitempty"`

	// TillerVersionDeprecated is deprecated in Helm 3, and therefore ignored. However, with a strict
	// YAML parser enabled, this field must be present.
	TillerVersionDeprecated string `yaml:"tillerVersion,omitempty"`

	// URLDeprecated is deprecated in Helm 3, superseded by URLs. It is ignored. However,
	// with a strict YAML parser enabled, this must be present on the struct.
	URLDeprecated string `yaml:"url,omitempty"`
}

func GetUrlFromRemoteIndex(chartName string) (string, error) {
	rsp, err := http.Get(defaultIndexURL)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	body, _ := ioutil.ReadAll(rsp.Body)

	var index IndexFile
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
