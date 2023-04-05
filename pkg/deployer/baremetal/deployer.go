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

package baremetal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type deployer struct {
	logger logger.Logger
	config *Config
	am     *ArtifactManager
	wg     sync.WaitGroup

	workingDir string
	logsDir    string
	pidsDir    string
	dataDir    string
}

var _ Deployer = &deployer{}

type Option func(*deployer)

func NewDeployer(l logger.Logger, clusterName string, opts ...Option) (Deployer, error) {
	d := &deployer{
		logger: l,
		config: defaultConfig(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	if err := d.config.Validate(); err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	d.workingDir = path.Join(homeDir, GtctlDir)
	if err := utils.CreateDirIfNotExists(d.workingDir); err != nil {
		return nil, err
	}

	am, err := NewArtifactManager(d.workingDir, l, false)
	if err != nil {
		return nil, err
	}
	d.am = am

	if err := d.createClusterDirs(clusterName); err != nil {
		return nil, err
	}

	return d, nil
}

func WithConfig(config *Config) Option {
	// TODO(zyy17): Should merge the default configuration.
	return func(d *deployer) {
		d.config = config
	}
}

func (d *deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *deployer) ListGreptimeDBClusters(ctx context.Context, options *ListGreptimeDBClustersOptions) ([]*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateGreptimeDBCluster(ctx context.Context, clusterName string, options *CreateGreptimeDBClusterOptions) error {
	if err := d.am.PrepareArtifact(GreptimeArtifactType, d.config.Cluster.Artifact); err != nil {
		return err
	}

	binary, err := d.am.BinaryPath(GreptimeArtifactType, d.config.Cluster.Artifact)
	if err != nil {
		return err
	}

	if err := d.startMetasrv(binary); err != nil {
		return err
	}

	if err := d.startDatanodes(binary, d.config.Cluster.Datanode.Replicas); err != nil {
		return err
	}

	if err := d.startFrontend(binary); err != nil {
		return err
	}

	return nil
}

func (d *deployer) UpdateGreptimeDBCluster(ctx context.Context, name string, options *UpdateGreptimeDBClusterOptions) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) DeleteGreptimeDBCluster(ctx context.Context, name string, options *DeleteGreptimeDBClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateEtcdCluster(ctx context.Context, clusterName string, options *CreateEtcdClusterOptions) error {
	if err := d.am.PrepareArtifact(EtcdArtifactType, d.config.Etcd.Artifact); err != nil {
		return err
	}

	bin, err := d.am.BinaryPath(EtcdArtifactType, d.config.Etcd.Artifact)
	if err != nil {
		return err
	}

	var (
		etcdDataDir = path.Join(d.dataDir, "etcd")
		etcdLogDir  = path.Join(d.logsDir, "etcd")
		etcdPidDir  = path.Join(d.pidsDir, "etcd")
		etcdDirs    = []string{etcdDataDir, etcdLogDir, etcdPidDir}
	)
	for _, dir := range etcdDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}

	args := []string{"--data-dir", etcdDataDir}
	go func() {
		err := d.runBinary(bin, args, etcdLogDir, etcdPidDir)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func (d *deployer) DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error {
	// We don't need to implement this method because we don't need to deploy GreptimeDB Operator.
	return fmt.Errorf("only support for k8s deployer")
}

func (d *deployer) Wait(ctx context.Context) error {
	d.wg.Wait()
	return nil
}

func (d *deployer) createClusterDirs(clusterName string) error {
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}/logs.
		logsDir = path.Join(d.workingDir, clusterName, "logs")

		// ${HOME}/${GtctlDir}/${ClusterName}/data.
		dataDir = path.Join(d.workingDir, clusterName, "data")

		// ${HOME}/${GtctlDir}/${ClusterName}/pids.
		pidsDir = path.Join(d.workingDir, clusterName, "pids")
	)

	dirs := []string{
		// ${HOME}/${GtctlDir}/${ClusterName}.
		path.Join(d.workingDir, clusterName),

		logsDir,
		dataDir,
		pidsDir,
	}

	for _, dir := range dirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}

	d.logsDir = logsDir
	d.dataDir = dataDir
	d.pidsDir = pidsDir

	return nil
}

func (d *deployer) runBinary(binary string, args []string, logDir string, pidDir string) error {
	cmd := exec.Command(binary, args...)

	// output to binary.
	logFile := path.Join(logDir, "log")
	outputFile, err := os.Create(logFile)
	if err != nil {
		return err
	}

	outputFileWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = outputFileWriter
	cmd.Stderr = outputFileWriter

	if err := cmd.Start(); err != nil {
		return err
	}

	pidFile := path.Join(pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(strconv.Itoa(cmd.Process.Pid)))
	if err != nil {
		return err
	}

	go func() {
		defer d.wg.Done()
		d.wg.Add(1)
		if err := cmd.Wait(); err != nil {
			panic(err)
		}
	}()

	return nil
}

func (d *deployer) startMetasrv(binary string) error {
	var (
		metasrvLogDir = path.Join(d.logsDir, "metasrv")
		metasrvPidDir = path.Join(d.pidsDir, "metasrv")
		metasrvDirs   = []string{metasrvLogDir, metasrvPidDir}
	)
	for _, dir := range metasrvDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}

	if err := d.runBinary(binary, d.buildMetasrvArgs(), metasrvLogDir, metasrvPidDir); err != nil {
		return err
	}

	// FIXME(zyy17): Wait for the metasrv to start. The datanode will fail to start if the metasrv is not ready.
	time.Sleep(2 * time.Second)

	return nil
}

func (d *deployer) startDatanodes(binary string, datanodeNum int) error {
	for i := 0; i < datanodeNum; i++ {
		dirName := fmt.Sprintf("datanode.%d", i)

		datanodeLogDir := path.Join(d.logsDir, dirName)
		if err := utils.CreateDirIfNotExists(datanodeLogDir); err != nil {
			return err
		}

		datanodePidDir := path.Join(d.pidsDir, dirName)
		if err := utils.CreateDirIfNotExists(datanodePidDir); err != nil {
			return err
		}

		datanodeDataDir := path.Join(d.dataDir, dirName)
		if err := utils.CreateDirIfNotExists(datanodeDataDir); err != nil {
			return err
		}

		if err := d.runBinary(binary, d.buildDatanodeArgs(i, datanodeDataDir), datanodeLogDir, datanodePidDir); err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) startFrontend(binary string) error {
	var (
		frontendLogDir = path.Join(d.logsDir, "frontend")
		frontendPidDir = path.Join(d.pidsDir, "frontend")
		frontendDirs   = []string{frontendLogDir, frontendPidDir}
	)
	for _, dir := range frontendDirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}

	if err := d.runBinary(binary, d.buildFrontendArgs(), frontendLogDir, frontendPidDir); err != nil {
		return err
	}

	return nil
}

func (d *deployer) buildMetasrvArgs() []string {
	args := []string{
		"metasrv", "start",
		"--store-addr", d.config.Cluster.Meta.StoreAddr,
		"--server-addr", d.config.Cluster.Meta.ServerAddr,
	}
	return args
}

func (d *deployer) buildDatanodeArgs(nodeID int, dataDir string) []string {
	rpcPort := d.config.Cluster.Datanode.RPCPort + nodeID
	mysqlPort := d.config.Cluster.Datanode.MySQLPort + nodeID
	args := []string{
		"datanode", "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
		fmt.Sprintf("--rpc-addr=0.0.0.0:%d", rpcPort),
		fmt.Sprintf("--mysql-addr=0.0.0.0:%d", mysqlPort),
		fmt.Sprintf("--data-dir=%s", path.Join(dataDir, "data")),
		fmt.Sprintf("--wal-dir=%s", path.Join(dataDir, "wal")),
	}
	return args
}

func (d *deployer) buildFrontendArgs() []string {
	args := []string{
		"frontend", "start",
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
	}
	return args
}
