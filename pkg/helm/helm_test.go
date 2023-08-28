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
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/google/go-cmp/cmp"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	testChartName      = "greptimedb"
	testChartsCacheDir = "/tmp/gtctl-test"
)

func TestLoadAndRenderChart(t *testing.T) {
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

	tests := []struct {
		name         string
		namespace    string
		chartName    string
		chartVersion string
		values       interface{}
	}{
		{
			name:         "greptimedb",
			namespace:    "default",
			chartName:    GreptimeDBChartName,
			chartVersion: "", // latest
			values:       deployer.CreateGreptimeDBClusterOptions{},
		},
		{
			name:         "greptimedb-operator",
			namespace:    "default",
			chartName:    GreptimeDBOperatorChartName,
			chartVersion: "", // latest
			values:       deployer.CreateGreptimeDBOperatorOptions{},
		},
		{
			name:         "etcd",
			namespace:    "default",
			chartName:    EtcdBitnamiOCIRegistry,
			chartVersion: DefaultEtcdChartVersion,
			values:       deployer.CreateGreptimeDBOperatorOptions{},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifests, err := r.LoadAndRenderChart(ctx, tt.name, tt.namespace, tt.chartName, tt.chartVersion, false, tt.values)
			if err != nil {
				t.Errorf("failed to load and render chart: %v", err)
			}
			if len(manifests) == 0 {
				t.Errorf("expected manifests to be non-empty")
			}
		})
	}
}

func TestRender_GetIndexFile(t *testing.T) {
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

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
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, err := r.getIndexFile(context.Background(), tt.url)
			if err != nil {
				t.Errorf("fetch index '%s' failed, err: %v", tt.url, err)
			}
		})
	}
}

func TestRender_GetLatestChartLatestChart(t *testing.T) {
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

	tests := []struct {
		url string
	}{
		{
			url: "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			indexFile, err := r.getIndexFile(context.Background(), tt.url)
			if err != nil {
				t.Errorf("fetch index '%s' failed, err: %v", tt.url, err)
			}

			chart, err := r.getLatestChart(indexFile, testChartName)
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
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

	options := deployer.CreateGreptimeDBClusterOptions{
		GreptimeDBChartVersion:      "",
		ImageRegistry:               "registry.cn-hangzhou.aliyuncs.com",
		DatanodeStorageClassName:    "ebs-sc",
		DatanodeStorageSize:         "11Gi",
		DatanodeStorageRetainPolicy: "Delete",
		EtcdEndPoint:                "127.0.0.1:2379",
		InitializerImageRegistry:    "registry.cn-hangzhou.aliyuncs.com",
		ConfigValues:                "meta.replicas=4",
	}

	values, err := r.generateHelmValues(options)
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
		"meta.replicas=4",
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
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

	options := deployer.CreateGreptimeDBOperatorOptions{
		GreptimeDBOperatorChartVersion: "",
		ImageRegistry:                  "registry.cn-hangzhou.aliyuncs.com",
		ConfigValues:                   "replicas=3",
	}

	values, err := r.generateHelmValues(options)
	if err != nil {
		t.Errorf("generate greptimedb operator helm values failed, err: %v", err)
	}

	ArgsStr := []string{
		"image.registry=registry.cn-hangzhou.aliyuncs.com",
		"replicas=3",
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
	r, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()),
		WithChartsCacheDir(testChartsCacheDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanChartsCache()

	options := deployer.CreateEtcdClusterOptions{
		EtcdChartVersion:     "",
		ImageRegistry:        "registry.cn-hangzhou.aliyuncs.com",
		EtcdStorageClassName: "ebs-sc",
		EtcdStorageSize:      "11Gi",
		EtcdClusterSize:      "3",
		ConfigValues:         "image.tag=latest",
	}

	values, err := r.generateHelmValues(options)
	if err != nil {
		t.Errorf("generate etcd helm values failed, err: %v", err)
	}

	ArgsStr := []string{
		"image.registry=registry.cn-hangzhou.aliyuncs.com",
		"persistence.storageClass=ebs-sc",
		"persistence.size=11Gi",
		"replicaCount=3",
		"image.tag=latest",
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

func cleanChartsCache() {
	os.RemoveAll(testChartsCacheDir)
}
