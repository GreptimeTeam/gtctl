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
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type Deployer struct {
	logger logger.Logger
	config *Config
	am     *ArtifactManager
	wg     sync.WaitGroup

	workingDir string
	logsDir    string
	pidsDir    string
	dataDir    string

	alwaysDownload bool
}

var _ Interface = &Deployer{}

type Option func(*Deployer)

func NewDeployer(l logger.Logger, clusterName string, opts ...Option) (Interface, error) {
	d := &Deployer{
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

	am, err := NewArtifactManager(d.workingDir, l, d.alwaysDownload)
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
	return func(d *Deployer) {
		d.config = config
	}
}

func WithGreptimeVersion(version string) Option {
	return func(d *Deployer) {
		d.config.Cluster.Artifact.Version = version
	}
}

func WithAlawaysDownload(alwaysDownload bool) Option {
	return func(d *Deployer) {
		d.alwaysDownload = alwaysDownload
	}
}

func (d *Deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *Deployer) ListGreptimeDBClusters(ctx context.Context, options *ListGreptimeDBClustersOptions) ([]*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *Deployer) CreateGreptimeDBCluster(ctx context.Context, clusterName string, options *CreateGreptimeDBClusterOptions) error {
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

func (d *Deployer) UpdateGreptimeDBCluster(ctx context.Context, name string, options *UpdateGreptimeDBClusterOptions) error {
	return fmt.Errorf("unsupported operation")
}

func (d *Deployer) DeleteGreptimeDBCluster(ctx context.Context, name string, options *DeleteGreptimeDBClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *Deployer) CreateEtcdCluster(ctx context.Context, clusterName string, options *CreateEtcdClusterOptions) error {
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

	if err := d.runBinary(bin, d.buildEtcdArgs(etcdDataDir), etcdLogDir, etcdPidDir); err != nil {
		return err
	}

	if err := d.checkEtcdHealth(bin); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) checkEtcdHealth(etcdBin string) error {
	// It's very likely that "etcdctl" is under the same directory of "etcd".
	etcdctlBin := path.Join(etcdBin, "../etcdctl")
	exists, err := utils.IsFileExists(etcdctlBin)
	if err != nil {
		return err
	}
	if !exists {
		d.logger.V(3).Infof("'etcdctl' is not found under the same directory of 'etcd', skip checking the healthy of Etcd.")
		return nil
	}

	for retry := 0; retry < 10; retry++ {
		outputRaw, err := exec.Command(etcdctlBin, "endpoint", "status").Output()
		if err != nil {
			return err
		}
		output := string(outputRaw)
		statuses := strings.Split(output, "\n")

		hasLeader := false
		for i := 0; i < len(statuses); i++ {
			fields := strings.Split(statuses[i], ",")

			// We are checking Etcd status with default output format("--write-out=simple"), example output:
			// 127.0.0.1:2379, 8e9e05c52164694d, 3.5.0, 131 kB, true, false, 3, 75, 75,
			//
			// The output fields are corresponding to the following table's columns (with format "--write-out=table"):
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			// |    ENDPOINT    |        ID        | VERSION | DB SIZE | IS LEADER | IS LEARNER | RAFT TERM | RAFT INDEX | RAFT APPLIED INDEX | ERRORS |
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			// | 127.0.0.1:2379 | 8e9e05c52164694d |   3.5.0 |  131 kB |      true |      false |         3 |         72 |                 72 |        |
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			//
			// So we can just check the "IS LEADER" field.
			if strings.TrimSpace(fields[4]) == "true" {
				hasLeader = true
				break
			}
		}
		if hasLeader {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Etcd is not ready in 10 second! You can find its logs in %s", path.Join(d.logsDir, "etcd"))
}

func (d *Deployer) DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *Deployer) CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error {
	// We don't need to implement this method because we don't need to deploy GreptimeDB Operator.
	return fmt.Errorf("only support for k8s Deployer")
}

func (d *Deployer) Wait(ctx context.Context) error {
	d.wg.Wait()
	return nil
}

func (d *Deployer) Config() *Config {
	return d.config
}

func (d *Deployer) createClusterDirs(clusterName string) error {
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

func (d *Deployer) runBinary(binary string, args []string, logDir string, pidDir string) error {
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

	d.logger.V(3).Infof("run binary '%s' with args: '%v', log: '%s', pid: '%s'", binary, args, logDir, pidDir)

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
			d.logger.Errorf("binary '%s' exited with error: %v", binary, err)
		}
	}()

	return nil
}

func (d *Deployer) startMetasrv(binary string) error {
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

	// FIXME(zyy17): Should add a timeout here.
	for {
		if d.isMetasrvRunning() {
			break
		}
	}

	return nil
}

func (d *Deployer) isMetasrvRunning() bool {
	_, httpPort, err := net.SplitHostPort(d.config.Cluster.Meta.HTTPAddr)
	if err != nil {
		d.logger.V(5).Infof("failed to split host port: %s", err)
		return false
	}

	rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
	if err != nil {
		d.logger.V(5).Infof("failed to get metasrv health: %s", err)
		return false
	}
	defer rsp.Body.Close()

	return rsp.StatusCode == http.StatusOK
}

func (d *Deployer) startDatanodes(binary string, datanodeNum int) error {
	dataHome := path.Join(d.dataDir, "home")
	if err := utils.CreateDirIfNotExists(dataHome); err != nil {
		return err
	}

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

		walDir := path.Join(d.dataDir, dirName, "wal")
		if err := utils.CreateDirIfNotExists(walDir); err != nil {
			return err
		}

		if err := d.runBinary(binary, d.buildDatanodeArgs(i, dataHome, walDir), datanodeLogDir, datanodePidDir); err != nil {
			return err
		}
	}

	// FIXME(zyy17): Should add a timeout here.
	for {
		if d.isDatanodesRunning() {
			break
		}
	}

	return nil
}

func (d *Deployer) isDatanodesRunning() bool {
	for i := 0; i < d.config.Cluster.Datanode.Replicas; i++ {
		addr := d.generateDatanodeAddr(d.config.Cluster.Datanode.HTTPAddr, i)
		_, httpPort, err := net.SplitHostPort(addr)
		if err != nil {
			d.logger.V(5).Infof("failed to split host port: %s", err)
			return false
		}

		rsp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
		if err != nil {
			d.logger.V(5).Infof("failed to get datanode health: %s", err)
			return false
		}
		defer rsp.Body.Close()

		if rsp.StatusCode != http.StatusOK {
			return false
		}
	}

	return true
}

func (d *Deployer) startFrontend(binary string) error {
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

func (d *Deployer) buildEtcdArgs(dataDir string) []string {
	return []string{"--data-dir", dataDir}
}

func (d *Deployer) buildMetasrvArgs() []string {
	args := []string{
		"metasrv", "start",
		"--store-addr", d.config.Cluster.Meta.StoreAddr,
		"--server-addr", d.config.Cluster.Meta.ServerAddr,
		"--http-addr", d.config.Cluster.Meta.HTTPAddr,
	}
	return args
}

func (d *Deployer) buildDatanodeArgs(nodeID int, dataHome string, walDir string) []string {
	args := []string{
		"datanode", "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
		fmt.Sprintf("--rpc-addr=%s", d.generateDatanodeAddr(d.config.Cluster.Datanode.RPCAddr, nodeID)),
		fmt.Sprintf("--http-addr=%s", d.generateDatanodeAddr(d.config.Cluster.Datanode.HTTPAddr, nodeID)),
		fmt.Sprintf("--data-home=%s", dataHome),
		fmt.Sprintf("--wal-dir=%s", walDir),
	}
	return args
}

func (d *Deployer) buildFrontendArgs() []string {
	args := []string{
		"frontend", "start",
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
	}
	return args
}

// TODO(zyy17): We can support port range in the future.
func (d *Deployer) generateDatanodeAddr(addr string, nodeID int) string {
	// Already checked in validation.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeID))
}
