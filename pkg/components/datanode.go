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

package components

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	greptimev1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"

	"github.com/GreptimeTeam/gtctl/pkg/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

const (
	dataHomeDir = "home"
	dataWalDir  = "wal"
)

type datanode struct {
	config      *config.Datanode
	metaSrvAddr string

	workingDirs WorkingDirs
	wg          *sync.WaitGroup
	logger      logger.Logger

	dataHomeDirs []string
	allocatedDirs
}

func NewDataNode(config *config.Datanode, metaSrvAddr string, workingDirs WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) ClusterComponent {
	return &datanode{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (d *datanode) Name() string {
	return string(greptimev1alpha1.DatanodeComponentKind)
}

func (d *datanode) Start(ctx context.Context, stop context.CancelFunc, binary string) error {
	for i := 0; i < d.config.Replicas; i++ {
		dirName := fmt.Sprintf("%s.%d", d.Name(), i)

		homeDir := path.Join(d.workingDirs.DataDir, dirName, dataHomeDir)
		if err := fileutils.EnsureDir(homeDir); err != nil {
			return err
		}
		d.dataHomeDirs = append(d.dataHomeDirs, homeDir)

		datanodeLogDir := path.Join(d.workingDirs.LogsDir, dirName)
		if err := fileutils.EnsureDir(datanodeLogDir); err != nil {
			return err
		}
		d.logsDirs = append(d.logsDirs, datanodeLogDir)

		datanodePidDir := path.Join(d.workingDirs.PidsDir, dirName)
		if err := fileutils.EnsureDir(datanodePidDir); err != nil {
			return err
		}
		d.pidsDirs = append(d.pidsDirs, datanodePidDir)

		walDir := path.Join(d.workingDirs.DataDir, dirName, dataWalDir)
		if err := fileutils.EnsureDir(walDir); err != nil {
			return err
		}
		d.dataDirs = append(d.dataDirs, path.Join(d.workingDirs.DataDir, dirName))

		option := &RunOptions{
			Binary: binary,
			Name:   dirName,
			logDir: datanodeLogDir,
			pidDir: datanodePidDir,
			args:   d.BuildArgs(i, walDir, homeDir),
		}
		if err := runBinary(ctx, stop, option, d.wg, d.logger); err != nil {
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

func (d *datanode) BuildArgs(params ...interface{}) []string {
	logLevel := d.config.LogLevel
	if logLevel == "" {
		logLevel = DefaultLogLevel
	}

	nodeID_, _, homeDir := params[0], params[1], params[2]
	nodeID := nodeID_.(int)

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		d.Name(), "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.metaSrvAddr),
		fmt.Sprintf("--rpc-addr=%s", generateDatanodeAddr(d.config.RPCAddr, nodeID)),
		fmt.Sprintf("--http-addr=%s", generateDatanodeAddr(d.config.HTTPAddr, nodeID)),
		fmt.Sprintf("--data-home=%s", homeDir),
	}

	if len(d.config.Config) > 0 {
		args = append(args, fmt.Sprintf("-c=%s", d.config.Config))
	}

	return args
}

func (d *datanode) IsRunning(_ context.Context) bool {
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

func generateDatanodeAddr(addr string, nodeID int) string {
	// Already checked in validation.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeID))
}
