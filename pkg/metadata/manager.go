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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

// Manager is the interface of the metadata manager.
// The metadata manager is responsible for managing all the metadata of gtctl.
type Manager interface {
	// AllocateArtifactFilePath allocates the file path of the artifact.
	AllocateArtifactFilePath(src *artifacts.Source, installBinary bool) (string, error)

	// AllocateClusterScopeDirs allocates the directories of one cluster.
	AllocateClusterScopeDirs() error

	// AllocateClusterConfigPath allocates the config path of one cluster.
	AllocateClusterConfigPath(cfg *config.Config) error

	// SetHomeDir sets the home directory of the metadata manager.
	SetHomeDir(dir string) error

	// GetWorkingDir returns the working directory of the metadata manager.
	// It should be ${HomeDir}/${BaseDir}.
	GetWorkingDir() string

	// GetClusterScopeDir returns the cluster scope directory of current cluster.
	GetClusterScopeDir() *ClusterScopeDir

	// Clean cleans up all the metadata. It will remove the working directory.
	Clean() error
}

const (
	// BaseDir is the working directory of gtctl and
	// all the metadata will be stored in ${HomeDir}/${BaseDir}.
	BaseDir = ".gtctl"

	// Default cluster scope directories.
	clusterLogsDir = "logs"
	clusterDataDir = "data"
	clusterPidsDir = "pids"
)

type ClusterScopeDir struct {
	BaseDir    string
	LogsDir    string
	DataDir    string
	PidsDir    string
	ConfigPath string
}

type manager struct {
	workingDir string

	clusterDir *ClusterScopeDir
}

var _ Manager = &manager{}

func New(homeDir string, clusterName string) (Manager, error) {
	m := &manager{}
	if homeDir == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		m.workingDir = filepath.Join(dir, BaseDir)
	} else {
		m.workingDir = filepath.Join(homeDir, BaseDir)
	}

	if len(clusterName) > 0 {
		m.initClusterScopeDirs(clusterName)
	}

	return m, nil
}

func (m *manager) initClusterScopeDirs(clusterName string) {
	csd := &ClusterScopeDir{
		// ${HomeDir}/${BaseDir}${ClusterName}
		BaseDir: path.Join(m.workingDir, clusterName),
	}

	// ${HomeDir}/${BaseDir}/${ClusterName}/logs
	csd.LogsDir = path.Join(csd.BaseDir, clusterLogsDir)
	// ${HomeDir}/${BaseDir}/${ClusterName}/data
	csd.DataDir = path.Join(csd.BaseDir, clusterDataDir)
	// ${HomeDir}/${BaseDir}/${ClusterName}/pids
	csd.PidsDir = path.Join(csd.BaseDir, clusterPidsDir)
	// ${HomeDir}/${BaseDir}/${ClusterName}/${ClusterName}.yaml
	csd.ConfigPath = filepath.Join(csd.BaseDir, fmt.Sprintf("%s.yaml", clusterName))

	m.clusterDir = csd
}

func (m *manager) AllocateArtifactFilePath(src *artifacts.Source, installBinary bool) (string, error) {
	var filePath string
	switch src.Type {
	case artifacts.ArtifactTypeChart:
		filePath = filepath.Join(m.workingDir, "artifacts", "charts", src.Name, src.Version, "pkg")
	case artifacts.ArtifactTypeBinary:
		if installBinary {
			// TODO(zyy17): It seems that we need to call AllocateArtifactFilePath() twice to get the correct path. Can we make it easier?
			filePath = filepath.Join(m.workingDir, "artifacts", "binaries", src.Name, src.Version, "bin")
		} else {
			filePath = filepath.Join(m.workingDir, "artifacts", "binaries", src.Name, src.Version, "pkg")
		}
	default:
		return "", fmt.Errorf("unknown artifact type: %s", src.Type)
	}

	return filePath, nil
}

func (m *manager) AllocateClusterScopeDirs() error {
	if m.clusterDir == nil {
		return fmt.Errorf("unallocated cluster dir, please initialize a metadata manager with cluster name provided")
	}

	dirs := []string{
		m.clusterDir.BaseDir,
		m.clusterDir.LogsDir,
		m.clusterDir.DataDir,
		m.clusterDir.PidsDir,
	}

	for _, dir := range dirs {
		if err := fileutils.EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) AllocateClusterConfigPath(cfg *config.Config) error {
	if m.clusterDir == nil {
		return fmt.Errorf("unallocated cluster dir, please initialize a metadata manager with cluster name provided")
	}

	if err := fileutils.EnsureDir(m.clusterDir.BaseDir); err != nil {
		return err
	}

	f, err := os.Create(m.clusterDir.ConfigPath)
	if err != nil {
		return err
	}

	metaConfig := config.RuntimeConfig{
		Config:        cfg,
		CreationDate:  time.Now(),
		ClusterDir:    m.clusterDir.BaseDir,
		ForegroundPid: os.Getpid(),
	}

	out, err := yaml.Marshal(metaConfig)
	if err != nil {
		return err
	}

	if _, err = f.Write(out); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	return nil
}

func (m *manager) SetHomeDir(dir string) error {
	m.workingDir = filepath.Join(dir, BaseDir)
	return nil
}

func (m *manager) GetWorkingDir() string {
	return m.workingDir
}

func (m *manager) GetClusterScopeDir() *ClusterScopeDir {
	return m.clusterDir
}

func (m *manager) Clean() error {
	return os.RemoveAll(m.workingDir)
}
