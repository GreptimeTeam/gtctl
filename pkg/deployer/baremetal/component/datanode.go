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
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type datanode struct {
	config      *config.Datanode
	metaSrvAddr string

	workingDirs WorkingDirs
	wg          *sync.WaitGroup
	logger      logger.Logger

	dataHome string
	allocatedDirs
}

func newDataNodes(config *config.Datanode, metaSrvAddr string, workingDirs WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &datanode{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (d *datanode) Name() string {
	return "datanode"
}

func (d *datanode) Start(ctx context.Context, binary string) error {
	dataHome := path.Join(d.workingDirs.DataDir, "home")
	if err := fileutils.CreateDirIfNotExists(dataHome); err != nil {
		return err
	}
	d.dataHome = dataHome

	for i := 0; i < d.config.Replicas; i++ {
		dirName := fmt.Sprintf("%s.%d", d.Name(), i)

		datanodeLogDir := path.Join(d.workingDirs.LogsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(datanodeLogDir); err != nil {
			return err
		}
		d.logsDirs = append(d.logsDirs, datanodeLogDir)

		datanodePidDir := path.Join(d.workingDirs.PidsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(datanodePidDir); err != nil {
			return err
		}
		d.pidsDirs = append(d.pidsDirs, datanodePidDir)

		walDir := path.Join(d.workingDirs.DataDir, dirName, "wal")
		if err := fileutils.CreateDirIfNotExists(walDir); err != nil {
			return err
		}
		d.dataDirs = append(d.dataDirs, path.Join(d.workingDirs.DataDir, dirName))

		option := &RunOptions{
			Binary: binary,
			Name:   dirName,
			logDir: datanodeLogDir,
			pidDir: datanodePidDir,
			args:   d.BuildArgs(ctx, i, walDir),
		}
		if err := runBinary(ctx, option, d.wg, d.logger); err != nil {
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

func (d *datanode) BuildArgs(ctx context.Context, params ...interface{}) []string {
	logLevel := d.config.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	nodeID_, walDir := params[0], params[1]
	nodeID := nodeID_.(int)

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		d.Name(), "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.metaSrvAddr),
		fmt.Sprintf("--rpc-addr=%s", generateDatanodeAddr(d.config.RPCAddr, nodeID)),
		fmt.Sprintf("--http-addr=%s", generateDatanodeAddr(d.config.HTTPAddr, nodeID)),
		fmt.Sprintf("--data-home=%s", d.dataHome),
		fmt.Sprintf("--wal-dir=%s", walDir),
	}
	return args
}

func (d *datanode) IsRunning(ctx context.Context) bool {
	for i := 0; i < d.config.Replicas; i++ {
		addr := generateDatanodeAddr(d.config.HTTPAddr, i)
		_, httpPort, err := net.SplitHostPort(addr)
		if err != nil {
			d.logger.V(5).Infof("failed to split host port in %s: %s", d.Name(), err)
			return false
		}

		rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
		if err != nil {
			d.logger.V(5).Infof("failed to get %s health: %s", d.Name(), err)
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

func (d *datanode) Delete(ctx context.Context, option DeleteOptions) error {
	if err := fileutils.DeleteDirIfExists(d.dataHome); err != nil {
		return err
	}

	if err := d.delete(ctx, option); err != nil {
		return err
	}

	return nil
}

func generateDatanodeAddr(addr string, nodeID int) string {
	// Already checked in validation.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeID))
}
