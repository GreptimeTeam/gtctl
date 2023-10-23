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
	"testing"

	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	testMetadataDir = "/tmp/gtctl-test"
)

func TestLoadAndRenderChart(t *testing.T) {
	r, err := NewLoader(logger.New(os.Stdout, log.Level(4), logger.WithColored()), WithHomeDir(testMetadataDir))
	if err != nil {
		t.Errorf("failed to create render: %v", err)
	}
	defer cleanMetadataDir()

	opts := &LoadOptions{
		ReleaseName:  "gtctl-ut",
		Namespace:    "default",
		ChartName:    artifacts.GreptimeDBClusterChartName,
		ChartVersion: "0.1.2",
		FromCNRegion: false,
		ValuesOptions: deployer.CreateGreptimeDBClusterOptions{
			ImageRegistry:               "registry.cn-hangzhou.aliyuncs.com",
			DatanodeStorageClassName:    "ebs-sc",
			DatanodeStorageSize:         "11Gi",
			DatanodeStorageRetainPolicy: "Delete",
			EtcdEndPoint:                "127.0.0.1:2379",
			InitializerImageRegistry:    "registry.cn-hangzhou.aliyuncs.com",
			ConfigValues:                "meta.replicas=3",
		},
		ValuesFile:  "./testdata/db-values.yaml",
		EnableCache: false,
	}

	ctx := context.Background()
	manifests, err := r.LoadAndRenderChart(ctx, opts)
	if err != nil {
		t.Fatalf("failed to load and render chart: %v", err)
	}

	wantedManifests, err := os.ReadFile("./testdata/db-manifests.yaml")
	if err != nil {
		t.Fatalf("failed to read wanted manifests: %v", err)
	}

	if string(wantedManifests) != string(manifests) {
		t.Errorf("expected %s, got %s", string(wantedManifests), string(manifests))
	}
}

func cleanMetadataDir() {
	os.RemoveAll(testMetadataDir)
}
