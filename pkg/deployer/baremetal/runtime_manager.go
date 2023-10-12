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

package baremetal

import (
	"fmt"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

const (
	defaultBaseDir = ".gtctl"
	defaultLogsDir = "logs"
	defaultDataDir = "data"
	defaultPidsDir = "pids"
)

// RuntimeManager manages all the runtime metadata used by cluster
// that running in bare-metal mode.
type RuntimeManager struct {
	clusterName string

	homeDir        string
	baseDir        string
	clusterDir     string
	clusterLogsDir string
	clusterDataDir string
	clusterPidsDir string

	clusterConfigPath string
}

func NewRuntimeManager(clusterName, homeDir string) *RuntimeManager {
	rm := &RuntimeManager{
		clusterName: clusterName,
		homeDir:     homeDir,
	}
	return rm
}

func (rm *RuntimeManager) configDirsAndPaths() {
	// ${HOME}/.gtctl
	rm.baseDir = path.Join(rm.homeDir, defaultBaseDir)
	// ${HOME}/.gtctl/${ClusterName}
	rm.clusterDir = path.Join(rm.baseDir, rm.clusterName)
	// ${HOME}/.gtctl/${ClusterName}/logs
	rm.clusterLogsDir = path.Join(rm.clusterDir, defaultLogsDir)
	// ${HOME}/.gtctl/${ClusterName}/data
	rm.clusterDataDir = path.Join(rm.clusterDir, defaultDataDir)
	// ${HOME}/.gtctl/${ClusterName}/pids
	rm.clusterPidsDir = path.Join(rm.clusterDir, defaultPidsDir)
	// ${HOME}/.gtctl/${ClusterName}/${ClusterName}.yaml
	rm.clusterConfigPath = path.Join(rm.clusterDir, fmt.Sprintf("%s.yaml", rm.clusterName))
}

func (rm *RuntimeManager) createDirs() error {
	dirs := []string{
		rm.baseDir,
		rm.clusterDir,
		rm.clusterLogsDir,
		rm.clusterDataDir,
		rm.clusterPidsDir,
	}

	for _, dir := range dirs {
		if err := fileutils.EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func (rm *RuntimeManager) createPaths(clusterConfig *config.Config) error {
	f, err := os.Create(rm.clusterConfigPath)
	if err != nil {
		return err
	}

	metaConfig := config.RuntimeConfig{
		Config:        clusterConfig,
		CreationDate:  time.Now(),
		ClusterDir:    rm.clusterDir,
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
