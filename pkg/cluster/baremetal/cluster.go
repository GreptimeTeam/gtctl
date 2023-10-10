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
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type Cluster struct {
	logger logger.Logger
	config *Config
	wg     sync.WaitGroup
	bm     *component.BareMetalCluster
	ctx    context.Context

	// TODO(sh2): move these dir into meta manager
	createNoDirs      bool
	workingDirs       component.WorkingDirs
	clusterDir        string
	baseDir           string
	clusterConfigPath string

	am artifacts.Manager
	mm metadata.Manager

	enableCache bool
}

type Option func(cluster *Cluster)

func NewCluster(l logger.Logger, clusterName string, opts ...Option) (*Cluster, error) {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	cluster := &Cluster{
		logger: l,
		config: DefaultConfig(),
		ctx:    ctx,
	}

	mm, err := metadata.New("")
	if err != nil {
		return nil, err
	}
	cluster.mm = mm

	for _, opt := range opts {
		if opt != nil {
			opt(cluster)
		}
	}

	if err = ValidateConfig(cluster.config); err != nil {
		return nil, err
	}

	if len(cluster.baseDir) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cluster.baseDir = path.Join(homeDir, GtctlDir)
	}

	if err = fileutils.EnsureDir(cluster.baseDir); err != nil {
		return nil, err
	}

	am, err := artifacts.NewManager(l)
	if err != nil {
		return nil, err
	}
	cluster.am = am

	cluster.initClusterDirsAndPath(clusterName)

	// TODO(sh2): implement it in the following PR
	// if !cluster.createNoDirs {}

	return cluster, nil
}

func (c *Cluster) initClusterDirsAndPath(clusterName string) {
	// Dirs
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}
		clusterDir = path.Join(c.baseDir, clusterName)

		// ${HOME}/${GtctlDir}/${ClusterName}/logs
		logsDir = path.Join(clusterDir, LogsDir)

		// ${HOME}/${GtctlDir}/${ClusterName}/data
		dataDir = path.Join(clusterDir, DataDir)

		// ${HOME}/${GtctlDir}/${ClusterName}/pids
		pidsDir = path.Join(clusterDir, PidsDir)
	)

	// Path
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}/${ClusterName}.yaml
		clusterConfigPath = path.Join(clusterDir, fmt.Sprintf("%s.yaml", clusterName))
	)

	c.clusterDir = clusterDir
	c.workingDirs = component.WorkingDirs{
		LogsDir: logsDir,
		DataDir: dataDir,
		PidsDir: pidsDir,
	}
	c.clusterConfigPath = clusterConfigPath
}
