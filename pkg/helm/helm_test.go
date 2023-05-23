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
	"sort"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/google/go-cmp/cmp"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/GreptimeTeam/gtctl/pkg/deployer"
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

func TestRender_GenerateGreptimeDBHelmValues(t *testing.T) {
	options := deployer.CreateGreptimeDBClusterOptions{
		GreptimeDBChartVersion:      "",
		ImageRegistry:               "registry.cn-hangzhou.aliyuncs.com",
		DatanodeStorageClassName:    "ebs-sc",
		DatanodeStorageSize:         "11Gi",
		DatanodeStorageRetainPolicy: "Delete",
		EtcdEndPoint:                "127.0.0.1:2379",
		InitializerImageRegistry:    "registry.cn-hangzhou.aliyuncs.com",
	}

	r := &Render{}
	values, err := r.GenerateHelmValues(options)
	if err != nil {
		t.Errorf("generate greptimedb helm values failed, err: %v", err)
	}

	ArgsStr := []string{
		"image.registry=registry.cn-hangzhou.aliyuncs.com",
		"datanode.storage.storageClassName=ebs-sc",
		"datanode.storage.storageSize=11Gi",
		"datanode.storage.storageRetainPolicy=Delete",
		"etcdEndpoints=127.0.0.1:2379",
		"initializer.registry=registry.cn-hangzhou.aliyuncs.com",
	}
	valuesWanted, err := strvals.Parse(strings.Join(ArgsStr, ","))
	if err != nil {
		t.Errorf("parse greptimedb helm values failed, err: %v", err)
	}

	if !cmp.Equal(values, valuesWanted) {
		t.Errorf("generate greptimedb helm values not match, expect: %v, got: %v", valuesWanted, values)
		t.Errorf("diff: %v", cmp.Diff(valuesWanted, values))
	}
}

func TestRender_GenerateGreptimeDBOperatorHelmValues(t *testing.T) {
	options := deployer.CreateGreptimeDBOperatorOptions{
		GreptimeDBOperatorChartVersion: "",
		ImageRegistry:                  "registry.cn-hangzhou.aliyuncs.com",
	}

	r := &Render{}
	values, err := r.GenerateHelmValues(options)
	if err != nil {
		t.Errorf("generate greptimedb operator helm values failed, err: %v", err)
	}

	ArgsStr := []string{
		"image.registry=registry.cn-hangzhou.aliyuncs.com",
	}
	valuesWanted, err := strvals.Parse(strings.Join(ArgsStr, ","))
	if err != nil {
		t.Errorf("parse greptimedb operator helm values failed, err: %v", err)
	}

	if !cmp.Equal(values, valuesWanted) {
		t.Errorf("generate greptimedb operator helm values not match, expect: %v, got: %v", valuesWanted, values)
		t.Errorf("diff: %v", cmp.Diff(valuesWanted, values))
	}
}

func TestRender_GenerateEtcdHelmValues(t *testing.T) {
	options := deployer.CreateEtcdClusterOptions{
		EtcdChartVersion:     "",
		ImageRegistry:        "registry.cn-hangzhou.aliyuncs.com",
		EtcdStorageClassName: "ebs-sc",
		EtcdStorageSize:      "11Gi",
	}

	r := &Render{}
	values, err := r.GenerateHelmValues(options)
	if err != nil {
		t.Errorf("generate etcd helm values failed, err: %v", err)
	}

	ArgsStr := []string{
		"image.registry=registry.cn-hangzhou.aliyuncs.com",
		"storage.storageClassName=ebs-sc",
		"storage.volumeSize=11Gi",
	}
	valuesWanted, err := strvals.Parse(strings.Join(ArgsStr, ","))
	if err != nil {
		t.Errorf("parse etcd helm values failed, err: %v", err)
	}

	if !cmp.Equal(values, valuesWanted) {
		t.Errorf("generate etcd helm values not match, expect: %v, got: %v", valuesWanted, values)
		t.Errorf("diff: %v", cmp.Diff(valuesWanted, values))
	}
}
