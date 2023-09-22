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

type metaSrv struct {
	config *config.MetaSrv

	workingDirs WorkingDirs
	wg          *sync.WaitGroup
	logger      logger.Logger

	allocatedDirs
}

func newMetaSrv(config *config.MetaSrv, workingDirs WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &metaSrv{
		config:      config,
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (m *metaSrv) Name() string {
	return MetaSrv
}

func (m *metaSrv) Start(ctx context.Context, binary string) error {
	// Default bind address for meta srv.
	bindAddr := net.JoinHostPort("127.0.0.1", "3002")
	if len(m.config.BindAddr) > 0 {
		bindAddr = m.config.BindAddr
	}

	for i := 0; i < m.config.Replicas; i++ {
		dirName := fmt.Sprintf("%s.%d", m.Name(), i)

		metaSrvLogDir := path.Join(m.workingDirs.LogsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(metaSrvLogDir); err != nil {
			return err
		}
		m.logsDirs = append(m.logsDirs, metaSrvLogDir)

		metaSrvPidDir := path.Join(m.workingDirs.PidsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(metaSrvPidDir); err != nil {
			return err
		}
		m.pidsDirs = append(m.pidsDirs, metaSrvPidDir)

		option := &RunOptions{
			Binary: binary,
			Name:   dirName,
			logDir: metaSrvLogDir,
			pidDir: metaSrvPidDir,
			args:   m.BuildArgs(ctx, i, bindAddr),
		}
		if err := runBinary(ctx, option, m.wg, m.logger); err != nil {
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
			if m.IsRunning(ctx) {
				break CHECKER
			}
		case <-ctx.Done():
			return fmt.Errorf("status checking failed: %v", ctx.Err())
		}
	}

	return nil
}

func (m *metaSrv) BuildArgs(ctx context.Context, params ...interface{}) []string {
	logLevel := m.config.LogLevel
	if logLevel == "" {
		logLevel = config.DefaultLogLevel
	}

	nodeID_, bindAddr_ := params[0], params[1]
	nodeID := nodeID_.(int)
	bindAddr := bindAddr_.(string)

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		m.Name(), "start",
		fmt.Sprintf("--store-addr=%s", m.config.StoreAddr),
		fmt.Sprintf("--server-addr=%s", m.config.ServerAddr),
		fmt.Sprintf("--http-addr=%s", generateMetaSrvAddr(m.config.HTTPAddr, nodeID)),
		fmt.Sprintf("--bind-addr=%s", generateMetaSrvAddr(bindAddr, nodeID)),
	}

	if len(m.config.Config) > 0 {
		args = append(args, fmt.Sprintf("-c %s", m.config.Config))
	}

	return args
}

func (m *metaSrv) IsRunning(ctx context.Context) bool {
	for i := 0; i < m.config.Replicas; i++ {
		addr := generateMetaSrvAddr(m.config.HTTPAddr, i)
		_, httpPort, err := net.SplitHostPort(addr)
		if err != nil {
			m.logger.V(5).Infof("failed to split host port in %s: %s", m.Name(), err)
			return false
		}

		rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
		if err != nil {
			m.logger.V(5).Infof("failed to get %s health: %s", m.Name(), err)
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

func (m *metaSrv) Delete(ctx context.Context, option DeleteOptions) error {
	if err := m.delete(ctx, option); err != nil {
		return err
	}
	return nil
}

func generateMetaSrvAddr(addr string, nodeID int) string {
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeID))
}
