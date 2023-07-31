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
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type dataNodes struct {
	config      *config.Datanode
	metaSrvAddr string

	dataDir string
	logsDir string
	pidsDir string
	wait    *sync.WaitGroup
	logger  logger.Logger

	dataHome         string
	dataNodeLogDirs  []string
	dataNodePidDirs  []string
	dataNodeDataDirs []string
}

func newDataNodes(config *config.Datanode, metaSrvAddr, dataDir, logsDir, pidsDir string,
	wait *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &dataNodes{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		dataDir:     dataDir,
		logsDir:     logsDir,
		pidsDir:     pidsDir,
		wait:        wait,
		logger:      logger,
	}
}

func (d *dataNodes) Start(ctx context.Context, binary string) error {
	dataHome := path.Join(d.dataDir, "home")
	if err := utils.CreateDirIfNotExists(dataHome); err != nil {
		return err
	}
	d.dataHome = dataHome

	for i := 0; i < d.config.Replicas; i++ {
		dirName := fmt.Sprintf("datanode.%d", i)

		datanodeLogDir := path.Join(d.logsDir, dirName)
		if err := utils.CreateDirIfNotExists(datanodeLogDir); err != nil {
			return err
		}
		d.dataNodeLogDirs = append(d.dataNodeLogDirs, datanodeLogDir)

		datanodePidDir := path.Join(d.pidsDir, dirName)
		if err := utils.CreateDirIfNotExists(datanodePidDir); err != nil {
			return err
		}
		d.dataNodePidDirs = append(d.dataNodePidDirs, datanodePidDir)

		walDir := path.Join(d.dataDir, dirName, "wal")
		if err := utils.CreateDirIfNotExists(walDir); err != nil {
			return err
		}
		d.dataNodeDataDirs = append(d.dataNodeDataDirs, path.Join(d.dataDir, dirName))

		if err := runBinary(ctx, binary, d.BuildArgs(ctx, i, walDir), datanodeLogDir, datanodePidDir, d.wait, d.logger); err != nil {
			return err
		}
	}

	// Checking component running status with intervals.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

CHECKER:
	for {
		select {
		case <-ticker.C:
			if d.IsRunning(ctx) {
				break CHECKER
			}
		case <-ctx.Done():
			return fmt.Errorf("status checking failed: %v", ctx.Err())
		}
	}

	return nil
}

func (d *dataNodes) BuildArgs(ctx context.Context, params ...interface{}) []string {
	logLevel := d.config.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	nodeID_, walDir := params[0], params[1]
	nodeID := nodeID_.(int)

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		"datanode", "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.metaSrvAddr),
		fmt.Sprintf("--rpc-addr=%s", generateDatanodeAddr(d.config.RPCAddr, nodeID)),
		fmt.Sprintf("--http-addr=%s", generateDatanodeAddr(d.config.HTTPAddr, nodeID)),
		fmt.Sprintf("--data-home=%s", d.dataHome),
		fmt.Sprintf("--wal-dir=%s", walDir),
	}
	return args
}

func (d *dataNodes) IsRunning(ctx context.Context) bool {
	for i := 0; i < d.config.Replicas; i++ {
		addr := generateDatanodeAddr(d.config.HTTPAddr, i)
		_, httpPort, err := net.SplitHostPort(addr)
		if err != nil {
			d.logger.V(5).Infof("failed to split host port: %s", err)
			return false
		}

		rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
		if err != nil {
			d.logger.V(5).Infof("failed to get datanode health: %s", err)
			return false
		}

		if rsp.StatusCode != http.StatusOK {
			return false
		}

		if err = rsp.Body.Close(); err != nil {
			return false
		}
	}

	return true
}

func (d *dataNodes) Delete(ctx context.Context) error {
	if err := utils.DeleteDirIfExists(d.dataHome); err != nil {
		return err
	}

	for _, dir := range d.dataNodeLogDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	for _, dir := range d.dataNodePidDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	for _, dir := range d.dataNodeDataDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	return nil
}

func generateDatanodeAddr(addr string, nodeID int) string {
	// Already checked in validation.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeID))
}
