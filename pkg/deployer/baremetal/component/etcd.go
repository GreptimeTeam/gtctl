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
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type etcd struct {
	dataDir string
	logsDir string
	pidsDir string
	wg      *sync.WaitGroup
	logger  logger.Logger

	etcdDirs []string
}

func newEtcd(dataDir, logsDir, pidsDir string,
	wg *sync.WaitGroup, logger logger.Logger) BareMetalClusterComponent {
	return &etcd{
		dataDir: dataDir,
		logsDir: logsDir,
		pidsDir: pidsDir,
		wg:      wg,
		logger:  logger,
	}
}

func (e *etcd) Start(ctx context.Context, binary string) error {
	var (
		etcdDataDir = path.Join(e.dataDir, "etcd")
		etcdLogDir  = path.Join(e.logsDir, "etcd")
		etcdPidDir  = path.Join(e.pidsDir, "etcd")
		etcdDirs    = []string{etcdDataDir, etcdLogDir, etcdPidDir}
	)
	for _, dir := range etcdDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	e.etcdDirs = etcdDirs

	if err := runBinary(ctx, binary, e.BuildArgs(ctx, etcdDataDir), etcdLogDir, etcdPidDir, e.wg, e.logger); err != nil {
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

func (e *etcd) Delete(ctx context.Context) error {
	for _, dir := range e.etcdDirs {
		if err := utils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}
	return nil
}
