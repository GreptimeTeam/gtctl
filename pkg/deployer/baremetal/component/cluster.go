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
	"syscall"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

// WorkingDirs include all the dirs used in bare-metal mode.
type WorkingDirs struct {
	DataDir string
	LogsDir string
	PidsDir string
}

type allocatedDirs struct {
	dataDirs []string
	logsDirs []string
	pidsDirs []string
}

type DeleteOptions struct {
	RetainLogs bool
}

// BareMetalCluster describes all the components need to be deployed under bare-metal mode.
type BareMetalCluster struct {
	MetaSrv  BareMetalClusterComponent
	Datanode BareMetalClusterComponent
	Frontend BareMetalClusterComponent
	Etcd     BareMetalClusterComponent
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
	Delete(ctx context.Context, option DeleteOptions) error

	// Name return the name of component.
	Name() string
}

func NewBareMetalCluster(config *config.Cluster, workingDirs WorkingDirs, wg *sync.WaitGroup, logger logger.Logger) *BareMetalCluster {
	return &BareMetalCluster{
		MetaSrv:  newMetaSrv(config.MetaSrv, workingDirs, wg, logger),
		Datanode: newDataNodes(config.Datanode, config.MetaSrv.ServerAddr, workingDirs, wg, logger),
		Frontend: newFrontend(config.Frontend, config.MetaSrv.ServerAddr, workingDirs, wg, logger),
		Etcd:     newEtcd(workingDirs, wg, logger),
	}
}

func runBinary(ctx context.Context, binary, name, logDir, pidDir string,
	args []string, wg *sync.WaitGroup, logger logger.Logger) error {
	cmd := exec.CommandContext(ctx, binary, args...)

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
		defer wg.Done()
		wg.Add(1)
		if err := cmd.Wait(); err != nil {
			// Caught signal kill and interrupt error then ignore.
			if exit, ok := err.(*exec.ExitError); ok {
				if status, ok := exit.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() &&
						(status.Signal() == syscall.SIGKILL || status.Signal() == syscall.SIGINT) {
						return
					}
				}
			}
			logger.Errorf("cluster component '%s' with binary '%s' exited: %v", name, binary, err)
		}
	}()

	return nil
}

func (ad *allocatedDirs) delete(ctx context.Context, option DeleteOptions) error {
	if !option.RetainLogs {
		for _, dir := range ad.logsDirs {
			if err := utils.DeleteDirIfExists(dir); err != nil {
				return err
			}
		}
	}

	for _, dir := range ad.dataDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	for _, dir := range ad.pidsDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	return nil
}
