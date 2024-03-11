/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package components

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"

	"github.com/GreptimeTeam/gtctl/pkg/config"
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

func NewFrontend(config *config.Frontend, metaSrvAddr string, workingDirs WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) ClusterComponent {
	return &frontend{
		config:      config,
		metaSrvAddr: metaSrvAddr,
		workingDirs: workingDirs,
		wg:          wg,
		logger:      logger,
	}
}

func (f *frontend) Name() string {
	return string(greptimedbclusterv1alpha1.FrontendComponentKind)
}

func (f *frontend) Start(ctx context.Context, stop context.CancelFunc, binary string) error {
	for i := 0; i < f.config.Replicas; i++ {
		dirName := fmt.Sprintf("%s.%d", f.Name(), i)

		frontendLogDir := path.Join(f.workingDirs.LogsDir, dirName)
		if err := fileutils.EnsureDir(frontendLogDir); err != nil {
			return err
		}
		f.logsDirs = append(f.logsDirs, frontendLogDir)

		frontendPidDir := path.Join(f.workingDirs.PidsDir, dirName)
		if err := fileutils.EnsureDir(frontendPidDir); err != nil {
			return err
		}
		f.pidsDirs = append(f.pidsDirs, frontendPidDir)

		option := &RunOptions{
			Binary: binary,
			Name:   dirName,
			logDir: frontendLogDir,
			pidDir: frontendPidDir,
			args:   f.BuildArgs(i),
		}
		if err := runBinary(ctx, stop, option, f.wg, f.logger); err != nil {
			return err
		}
	}

	return nil
}

func (f *frontend) BuildArgs(params ...interface{}) []string {
	logLevel := f.config.LogLevel
	if logLevel == "" {
		logLevel = DefaultLogLevel
	}

	nodeId := params[0].(int)

	args := []string{
		fmt.Sprintf("--log-level=%s", logLevel),
		f.Name(), "start",
		fmt.Sprintf("--metasrv-addr=%s", f.metaSrvAddr),
	}

	args = generateAddrArg("--http-addr", f.config.HTTPAddr, nodeId, args)
	args = generateAddrArg("--rpc-addr", f.config.GRPCAddr, nodeId, args)
	args = generateAddrArg("--mysql-addr", f.config.MysqlAddr, nodeId, args)
	args = generateAddrArg("--postgres-addr", f.config.PostgresAddr, nodeId, args)
	args = generateAddrArg("--opentsdb-addr", f.config.OpentsdbAddr, nodeId, args)

	if len(f.config.Config) > 0 {
		args = append(args, fmt.Sprintf("-c=%s", f.config.Config))
	}
	if len(f.config.UserProvider) > 0 {
		args = append(args, fmt.Sprintf("--user-provider=%s", f.config.UserProvider))
	}
	return args
}

func (f *frontend) IsRunning(_ context.Context) bool {
	for i := 0; i < f.config.Replicas; i++ {
		addr := formatAddrArg(f.config.HTTPAddr, i)
		healthy := fmt.Sprintf("http://%s/health", addr)

		resp, err := http.Get(healthy)
		if err != nil {
			f.logger.V(5).Infof("Failed to get %s healthy: %s", f.Name(), err)
			return false
		}

		if resp.StatusCode != http.StatusOK {
			f.logger.V(5).Infof("%s is not healthy: %s", f.Name(), resp)
			return false
		}

		if err = resp.Body.Close(); err != nil {
			f.logger.V(5).Infof("%s is not healthy: %s, err: %s", f.Name(), resp, err)
			return false
		}
	}
	return true
}

// formatAddrArg formats the given addr and nodeId to a valid socket string.
// This function will return an empty string when the given addr is empty.
func formatAddrArg(addr string, nodeId int) string {
	// return empty result if the address is not specified
	if len(addr) == 0 {
		return addr
	}

	// The "addr" is validated when set.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)

	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeId))
}

// generateAddrArg pushes arg into args array, return the new args array.
func generateAddrArg(config string, addr string, nodeId int, args []string) []string {
	socketAddr := formatAddrArg(addr, nodeId)

	// don't generate param if the socket address is empty
	if len(socketAddr) == 0 {
		return args
	}

	return append(args, fmt.Sprintf("%s=%s", config, socketAddr))
}
