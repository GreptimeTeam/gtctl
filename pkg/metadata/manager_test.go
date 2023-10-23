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

package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
)

func TestMetadataManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "gtctl-ut-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	m, err := New(tempDir, "")
	if err != nil {
		t.Fatalf("failed to create metadata manager: %v", err)
	}

	tests := []struct {
		src              *artifacts.Source
		wantedDestDir    string
		wantedInstallDir string
	}{
		{
			src: &artifacts.Source{
				Name:    artifacts.GreptimeDBChartName,
				Version: artifacts.LatestVersionTag,
				Type:    artifacts.ArtifactTypeChart,
			},
			wantedDestDir: filepath.Join(tempDir, BaseDir, "artifacts", "charts", artifacts.GreptimeDBChartName, artifacts.LatestVersionTag, "pkg"),
		},
		{
			src: &artifacts.Source{
				Name:    artifacts.EtcdBinName,
				Version: artifacts.DefaultEtcdBinVersion,
				Type:    artifacts.ArtifactTypeBinary,
			},
			wantedDestDir:    filepath.Join(tempDir, BaseDir, "artifacts", "binaries", artifacts.EtcdBinName, artifacts.DefaultEtcdBinVersion, "pkg"),
			wantedInstallDir: filepath.Join(tempDir, BaseDir, "artifacts", "binaries", artifacts.EtcdBinName, artifacts.DefaultEtcdBinVersion, "bin"),
		},
	}

	for _, tt := range tests {
		gotDestDir, err := m.AllocateArtifactFilePath(tt.src, false)
		if err != nil {
			t.Errorf("failed to allocate artifact file path: %v", err)
		}
		if gotDestDir != tt.wantedDestDir {
			t.Errorf("got %s, wanted %s", gotDestDir, tt.wantedDestDir)
		}

		if tt.src.Type == artifacts.ArtifactTypeBinary {
			gotInstallDir, err := m.AllocateArtifactFilePath(tt.src, true)
			if err != nil {
				t.Errorf("failed to allocate artifact file path: %v", err)
			}
			if gotInstallDir != tt.wantedInstallDir {
				t.Errorf("got %s, wanted %s", gotInstallDir, tt.wantedInstallDir)
			}
		}
	}

	// Clean() should remove the working directory.
	if err := m.Clean(); err != nil {
		t.Fatalf("failed to clean up metadata: %v", err)
	}

	if _, err := os.Stat(m.GetWorkingDir()); !os.IsNotExist(err) {
		t.Fatalf("working directory %s still exists", m.GetWorkingDir())
	}

	// SetHomeDir() should change the working directory.
	testHomeDir := "/path/to/gtctl-ut"
	if err := m.SetHomeDir(testHomeDir); err != nil {
		t.Fatalf("failed to set home directory: %v", err)
	}
	wantedWorkingDir := filepath.Join(testHomeDir, BaseDir)
	if m.GetWorkingDir() != wantedWorkingDir {
		t.Errorf("got %s, wanted %s", m.GetWorkingDir(), wantedWorkingDir)
	}
}

func TestCreateMetadataManagerWithEmptyHomeDir(t *testing.T) {
	m, err := New("", "")
	if err != nil {
		t.Fatalf("failed to create metadata manager: %v", err)
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home directory: %v", err)
	}

	wantedWorkingDir := filepath.Join(dir, BaseDir)
	if m.GetWorkingDir() != wantedWorkingDir {
		t.Fatalf("got %s, wanted %s", m.GetWorkingDir(), wantedWorkingDir)
	}
}

func TestMetadataManagerWithClusterConfigPath(t *testing.T) {
	m, err := New("/tmp", "test")
	assert.NoError(t, err)

	expect := config.DefaultConfig()
	err = m.AllocateClusterConfigPath(expect)
	assert.NoError(t, err)

	csd := m.GetClusterScopeDir()
	assert.NotNil(t, csd)
	assert.NotEmpty(t, csd.ConfigPath)

	cnt, err := os.ReadFile(csd.ConfigPath)
	assert.NoError(t, err)

	var actual config.RuntimeConfig
	err = yaml.Unmarshal(cnt, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expect, actual.Config)

	err = m.Clean()
	assert.NoError(t, err)
}
