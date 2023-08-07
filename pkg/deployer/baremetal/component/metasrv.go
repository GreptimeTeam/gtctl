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
	"sync"
	"time"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
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
	return "metasrv"
}

func (m *metaSrv) Start(ctx context.Context, binary string) error {
	var (
		metaSrvLogDir = path.Join(m.workingDirs.LogsDir, m.Name())
		metaSrvPidDir = path.Join(m.workingDirs.PidsDir, m.Name())
		metaSrvDirs   = []string{metaSrvLogDir, metaSrvPidDir}
	)
	for _, dir := range metaSrvDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	m.logsDirs = append(m.logsDirs, metaSrvLogDir)
	m.pidsDirs = append(m.pidsDirs, metaSrvPidDir)

	if err := runBinary(ctx, binary, m.Name(), metaSrvLogDir, metaSrvPidDir,
		m.BuildArgs(ctx), m.wg, m.logger); err != nil {
		return err
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
		logLevel = "info"
	}
	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		m.Name(), "start",
		"--store-addr", m.config.StoreAddr,
		"--server-addr", m.config.ServerAddr,
		"--http-addr", m.config.HTTPAddr,
	}
	return args
}

func (m *metaSrv) IsRunning(ctx context.Context) bool {
	_, httpPort, err := net.SplitHostPort(m.config.HTTPAddr)
	if err != nil {
		m.logger.V(5).Infof("failed to split host port in %s: %s", m.Name(), err)
		return false
	}

	rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
	if err != nil {
		m.logger.V(5).Infof("failed to get %s health: %s", m.Name(), err)
		return false
	}
	if err = rsp.Body.Close(); err != nil {
		return false
	}

	return rsp.StatusCode == http.StatusOK
}

func (m *metaSrv) Delete(ctx context.Context, option DeleteOptions) error {
	if err := m.delete(ctx, option); err != nil {
		return err
	}
	return nil
}
