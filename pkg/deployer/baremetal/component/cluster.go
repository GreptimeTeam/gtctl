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

package component

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

// BareMetalCluster describes all the components need to be deployed under bare-metal mode.
type BareMetalCluster struct {
	MetaSrv   BareMetalClusterComponent
	DataNodes BareMetalClusterComponent
	Frontend  BareMetalClusterComponent
	Etcd      BareMetalClusterComponent
}

// BareMetalClusterComponent is the basic unit of running GreptimeDB Cluster in bare-metal mode.
type BareMetalClusterComponent interface {
	// Start starts cluster component by executing binary.
	Start(ctx context.Context, binary string) error

	// BuildArgs build up args for cluster component.
	BuildArgs(ctx context.Context, params ...interface{}) []string

	// IsRunning returns the status of current cluster component.
	IsRunning(ctx context.Context) bool

	// Delete deletes resources that allocated in the system for current component.
	Delete(ctx context.Context) error
}

func NewGreptimeDBCluster(config *config.Cluster, dataDir, logsDir, pidsDir string,
	wait *sync.WaitGroup, logger logger.Logger) *BareMetalCluster {
	return &BareMetalCluster{
		MetaSrv:   newMetaSrv(config.Meta, logsDir, pidsDir, wait, logger),
		DataNodes: newDataNodes(config.Datanode, config.Meta.ServerAddr, dataDir, logsDir, pidsDir, wait, logger),
		Frontend:  newFrontend(config.Frontend, config.Meta.ServerAddr, logsDir, pidsDir, wait, logger),
		Etcd:      newEtcd(dataDir, logsDir, pidsDir, wait, logger),
	}
}

func runBinary(ctx context.Context, binary string, args []string, logDir string, pidDir string,
	wait *sync.WaitGroup, logger logger.Logger) error {
	cmd := exec.Command(binary, args...)

	// output to binary.
	logFile := path.Join(logDir, "log")
	outputFile, err := os.Create(logFile)
	if err != nil {
		return err
	}

	outputFileWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = outputFileWriter
	cmd.Stderr = outputFileWriter

	logger.V(3).Infof("run binary '%s' with args: '%v', log: '%s', pid: '%s'", binary, args, logDir, pidDir)

	if err := cmd.Start(); err != nil {
		return err
	}

	pidFile := path.Join(pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(strconv.Itoa(cmd.Process.Pid)))
	if err != nil {
		return err
	}

	go func() {
		defer wait.Done()
		wait.Add(1)
		// TODO(sh2) caught up the `signal: interrupt` error and ignore
		if err := cmd.Wait(); err != nil {
			logger.Errorf("binary '%s' exited with error: %v", binary, err)
		}
	}()

	return nil
}