// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package artifacts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

func TestDownloadCharts(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()))
	if err != nil {
		t.Fatalf("failed to create artifacts manager: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		version      string
		typ          ArtifactType
		fromCNRegion bool
	}{
		{GreptimeDBClusterChartName, "latest", ArtifactTypeChart, false},
		{GreptimeDBOperatorChartName, "latest", ArtifactTypeChart, false},
		{GreptimeDBClusterChartName, "0.1.2", ArtifactTypeChart, false},
		{GreptimeDBOperatorChartName, "0.1.1-alpha.12", ArtifactTypeChart, false},
		{EtcdChartName, DefaultEtcdChartVersion, ArtifactTypeChart, false},
	}
	for _, tt := range tests {
		src, err := m.NewSource(tt.name, tt.version, tt.typ, tt.fromCNRegion)
		if err != nil {
			t.Errorf("failed to create source: %v", err)
		}
		artifactFile, err := m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: false})
		if err != nil {
			t.Errorf("failed to download: %v", err)
		}

		_, err = os.Stat(artifactFile)
		if os.IsNotExist(err) {
			t.Errorf("artifact file does not exist: %v", err)
		}
		if err != nil {
			t.Errorf("failed to stat artifact file: %v", err)
		}
	}
}

func TestDownloadChartsFromCNRegion(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()))
	if err != nil {
		t.Fatalf("failed to create artifacts manager: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		version      string
		typ          ArtifactType
		fromCNRegion bool
	}{
		{GreptimeDBClusterChartName, LatestVersionTag, ArtifactTypeChart, true},
		{GreptimeDBOperatorChartName, LatestVersionTag, ArtifactTypeChart, true},
		{GreptimeDBClusterChartName, "0.1.2", ArtifactTypeChart, true},
		{GreptimeDBOperatorChartName, "0.1.1-alpha.12", ArtifactTypeChart, true},
		{EtcdChartName, DefaultEtcdChartVersion, ArtifactTypeChart, true},
	}
	for _, tt := range tests {
		src, err := m.NewSource(tt.name, tt.version, tt.typ, tt.fromCNRegion)
		if err != nil {
			t.Errorf("failed to create source: %v", err)
		}
		artifactFile, err := m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: false})
		if err != nil {
			t.Errorf("failed to download: %v", err)
		}

		_, err = os.Stat(artifactFile)
		if os.IsNotExist(err) {
			t.Errorf("artifact file does not exist: %v", err)
		}
		if err != nil {
			t.Errorf("failed to stat artifact file: %v", err)
		}
	}
}

func TestDownloadBinariesFromCNRegion(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()))
	if err != nil {
		t.Fatalf("failed to create artifacts manager: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		version      string
		typ          ArtifactType
		fromCNRegion bool
	}{
		{GreptimeBinName, "v0.4.0-nightly-20231002", ArtifactTypeBinary, true},
		{EtcdBinName, DefaultEtcdBinVersion, ArtifactTypeBinary, true},
	}
	for _, tt := range tests {
		src, err := m.NewSource(tt.name, tt.version, tt.typ, tt.fromCNRegion)
		if err != nil {
			t.Errorf("failed to create source: %v", err)
		}
		artifactFile, err := m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: false, BinaryInstallDir: filepath.Join(filepath.Dir(destDir(tempDir, src)), "bin")})
		if err != nil {
			t.Errorf("failed to download: %v", err)
		}

		info, err := os.Stat(artifactFile)
		if os.IsNotExist(err) {
			t.Errorf("artifact file does not exist: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("binary file is not executable")
		}
		if err != nil {
			t.Errorf("failed to stat artifact file: %v", err)
		}
	}
}

func TestDownloadBinaries(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()))
	if err != nil {
		t.Fatalf("failed to create artifacts manager: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		version      string
		typ          ArtifactType
		fromCNRegion bool
	}{
		{GreptimeBinName, LatestVersionTag, ArtifactTypeBinary, false},
		{GreptimeBinName, "v0.4.0-nightly-20231002", ArtifactTypeBinary, false},
		{EtcdBinName, DefaultEtcdBinVersion, ArtifactTypeBinary, false},
	}
	for _, tt := range tests {
		src, err := m.NewSource(tt.name, tt.version, tt.typ, tt.fromCNRegion)
		if err != nil {
			t.Errorf("failed to create source: %v", err)
		}
		artifactFile, err := m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: false, BinaryInstallDir: filepath.Join(filepath.Dir(destDir(tempDir, src)), "bin")})
		if err != nil {
			t.Errorf("failed to download: %v", err)
		}

		info, err := os.Stat(artifactFile)
		if os.IsNotExist(err) {
			t.Errorf("artifact file does not exist: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("binary file is not executable")
		}
		if err != nil {
			t.Errorf("failed to stat artifact file: %v", err)
		}
	}
}

func TestArtifactsCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewManager(logger.New(os.Stdout, log.Level(4), logger.WithColored()))
	if err != nil {
		t.Fatalf("failed to create artifacts manager: %v", err)
	}

	ctx := context.Background()

	src, err := m.NewSource(GreptimeDBClusterChartName, LatestVersionTag, ArtifactTypeChart, false)
	if err != nil {
		t.Errorf("failed to create source: %v", err)
	}
	artifactFile, err := m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: false})
	if err != nil {
		t.Errorf("failed to download: %v", err)
	}

	firstTimeInfo, err := os.Stat(artifactFile)
	if os.IsNotExist(err) {
		t.Errorf("artifact file does not exist: %v", err)
	}
	if err != nil {
		t.Errorf("failed to stat artifact file: %v", err)
	}

	// Download again with cache.
	artifactFile, err = m.DownloadTo(ctx, src, destDir(tempDir, src), &DownloadOptions{EnableCache: true})
	if err != nil {
		t.Errorf("failed to download: %v", err)
	}
	secondTimeInfo, err := os.Stat(artifactFile)
	if os.IsNotExist(err) {
		t.Errorf("artifact file does not exist: %v", err)
	}
	if err != nil {
		t.Errorf("failed to stat artifact file: %v", err)
	}
	if os.IsNotExist(err) {
		t.Errorf("artifact file does not exist: %v", err)
	}
	if err != nil {
		t.Errorf("failed to stat artifact file: %v", err)
	}

	if firstTimeInfo.ModTime() != secondTimeInfo.ModTime() {
		t.Errorf("artifact file is not cached")
	}
}

func destDir(workingDir string, src *Source) string {
	var artifactsDir string

	switch src.Type {
	case ArtifactTypeBinary:
		artifactsDir = "binaries"
	case ArtifactTypeChart:
		artifactsDir = "charts"
	default:
		panic(fmt.Sprintf("unknown artifact type: %s", src.Type))
	}

	return fmt.Sprintf("%s/artifacts/%s/%s/%s/pkg", workingDir, artifactsDir, src.Name, src.Version)
}
