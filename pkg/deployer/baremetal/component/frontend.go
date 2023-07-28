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
	"path"
	"sync"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type frontend struct {
	config      *config.Frontend
	metaSrvAddr string

	logsDir string
	pidsDir string
	wait    *sync.WaitGroup
	logger  logger.Logger

	frontendDirs []string
}

func newFrontend(config *config.Frontend, metaSrvAddr, logsDir, pidsDir string,
	wait *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &frontend{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		logsDir:     logsDir,
		pidsDir:     pidsDir,
		wait:        wait,
		logger:      logger,
	}
}

func (f *frontend) Start(ctx context.Context, binary string) error {
	var (
		frontendLogDir = path.Join(f.logsDir, "frontend")
		frontendPidDir = path.Join(f.pidsDir, "frontend")
		frontendDirs   = []string{frontendLogDir, frontendPidDir}
	)
	for _, dir := range frontendDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	f.frontendDirs = frontendDirs

	if err := runBinary(ctx, binary, f.BuildArgs(ctx), frontendLogDir, frontendPidDir, f.wait, f.logger); err != nil {
		return err
	}

	return nil
}

func (f *frontend) BuildArgs(ctx context.Context, params ...interface{}) []string {
	logLevel := f.config.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}
	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		"frontend", "start",
		fmt.Sprintf("--metasrv-addr=%s", f.metaSrvAddr),
	}
	return args
}

func (f *frontend) IsRunning(ctx context.Context) bool {
	// Have not implemented the healthy checker now.
	return false
}

func (f *frontend) Delete(ctx context.Context) error {
	for _, dir := range f.frontendDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}
	f.frontendDirs = nil
	return nil
}
