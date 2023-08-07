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
	"path"
	"sync"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type etcd struct {
	workingDirs WorkingDirs
	wg          *sync.WaitGroup
	logger      logger.Logger

	allocatedDirs
}

func newEtcd(workingDirs WorkingDirs, wg *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &etcd{
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (e *etcd) Name() string {
	return "etcd"
}

func (e *etcd) Start(ctx context.Context, binary string) error {
	var (
		etcdDataDir = path.Join(e.workingDirs.DataDir, e.Name())
		etcdLogDir  = path.Join(e.workingDirs.LogsDir, e.Name())
		etcdPidDir  = path.Join(e.workingDirs.PidsDir, e.Name())
		etcdDirs    = []string{etcdDataDir, etcdLogDir, etcdPidDir}
	)
	for _, dir := range etcdDirs {
		if err := fileutils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	e.dataDirs = append(e.dataDirs, etcdDataDir)
	e.logsDirs = append(e.logsDirs, etcdLogDir)
	e.pidsDirs = append(e.pidsDirs, etcdPidDir)

	if err := runBinary(ctx, binary, e.Name(), etcdLogDir, etcdPidDir,
		e.BuildArgs(ctx, etcdDataDir), e.wg, e.logger); err != nil {
		return err
	}

	return nil
}

func (e *etcd) BuildArgs(ctx context.Context, params ...interface{}) []string {
	return []string{"--data-dir", params[0].(string)}
}

func (e *etcd) IsRunning(ctx context.Context) bool {
	// Have not implemented the healthy checker now.
	return false
}

func (e *etcd) Delete(ctx context.Context, option DeleteOptions) error {
	if err := e.delete(ctx, option); err != nil {
		return err
	}
	return nil
}
