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
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type frontend struct {
	config      *config.Frontend
	metaSrvAddr string

	workingDirs WorkingDirs
	wg          *sync.WaitGroup
	logger      logger.Logger

	allocatedDirs
}

func newFrontend(config *config.Frontend, metaSrvAddr string, workingDirs WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &frontend{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (f *frontend) Name() string {
	return Frontend
}

func (f *frontend) Start(ctx context.Context, binary string) error {
	for i := 0; i < f.config.Replicas; i++ {
		dirName := fmt.Sprintf("%s.%d", f.Name(), i)

		frontendLogDir := path.Join(f.workingDirs.LogsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(frontendLogDir); err != nil {
			return err
		}
		f.logsDirs = append(f.logsDirs, frontendLogDir)

		frontendPidDir := path.Join(f.workingDirs.PidsDir, dirName)
		if err := fileutils.CreateDirIfNotExists(frontendPidDir); err != nil {
			return err
		}
		f.pidsDirs = append(f.pidsDirs, frontendPidDir)

		option := &RunOptions{
			Binary: binary,
			Name:   dirName,
			logDir: frontendLogDir,
			pidDir: frontendPidDir,
			args:   f.BuildArgs(ctx),
		}
		if err := runBinary(ctx, option, f.wg, f.logger); err != nil {
			return err
		}
	}

	return nil
}

func (f *frontend) BuildArgs(ctx context.Context, params ...interface{}) []string {
	logLevel := f.config.LogLevel
	if logLevel == "" {
		logLevel = config.DefaultLogLevel
	}

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		f.Name(), "start",
		fmt.Sprintf("--metasrv-addr=%s", f.metaSrvAddr),
	}

	if len(f.config.Config) > 0 {
		args = append(args, fmt.Sprintf("-c %s", f.config.Config))
	}

	return args
}

func (f *frontend) IsRunning(ctx context.Context) bool {
	// Have not implemented the healthy checker now.
	return false
}

func (f *frontend) Delete(ctx context.Context, option DeleteOptions) error {
	if err := f.delete(ctx, option); err != nil {
		return err
	}

	return nil
}
