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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

type Deployer struct {
	logger logger.Logger
	config *config.Config
	am     *ArtifactManager
	wg     sync.WaitGroup
	bm     *component.BareMetalCluster

	workingDir string
	clusterDir string
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
		config: config.DefaultConfig(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	if err := ValidateConfig(d.config); err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	d.workingDir = path.Join(homeDir, config.GtctlDir)
	if err := utils.CreateDirIfNotExists(d.workingDir); err != nil {
		return nil, err
	}

	am, err := NewArtifactManager(d.workingDir, l, d.alwaysDownload)
	if err != nil {
		return nil, err
	}
	d.am = am

	if len(clusterName) > 0 {
		if err := d.createClusterDirs(clusterName); err != nil {
			return nil, err
		}

		d.bm = component.NewGreptimeDBCluster(d.config.Cluster, d.dataDir, d.logsDir, d.pidsDir, &d.wg, d.logger)
	}

	return d, nil
}

func (d *Deployer) createClusterDirs(clusterName string) error {
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}
		clusterDir = path.Join(d.workingDir, clusterName)

		// ${HOME}/${GtctlDir}/${ClusterName}/logs.
		logsDir = path.Join(clusterDir, "logs")

		// ${HOME}/${GtctlDir}/${ClusterName}/data.
		dataDir = path.Join(clusterDir, "data")

		// ${HOME}/${GtctlDir}/${ClusterName}/pids.
		pidsDir = path.Join(clusterDir, "pids")
	)

	dirs := []string{
		clusterDir,
		logsDir,
		dataDir,
		pidsDir,
	}

	for _, dir := range dirs {
		if err := utils.CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}

	d.clusterDir = clusterDir
	d.logsDir = logsDir
	d.dataDir = dataDir
	d.pidsDir = pidsDir

	return nil
}

func WithConfig(config *config.Config) Option {
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
	if err := d.am.PrepareArtifact(ctx, GreptimeArtifactType, d.config.Cluster.Artifact); err != nil {
		return err
	}

	binary, err := d.am.BinaryPath(GreptimeArtifactType, d.config.Cluster.Artifact)
	if err != nil {
		return err
	}

	if err := d.bm.MetaSrv.Start(ctx, binary); err != nil {
		return err
	}

	if err := d.bm.DataNodes.Start(ctx, binary); err != nil {
		return err
	}

	if err := d.bm.Frontend.Start(ctx, binary); err != nil {
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

// deleteGreptimeDBClusterForeground delete the whole cluster if it runs in foreground.
func (d *Deployer) deleteGreptimeDBClusterForeground(ctx context.Context) error {
	// It is unnecessary to delete each component resources in cluster since it runs in the foreground.
	// So deleting the whole cluster resources here would be fine.
	if err := utils.DeleteDirIfExists(d.clusterDir); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) CreateEtcdCluster(ctx context.Context, clusterName string, options *CreateEtcdClusterOptions) error {
	if err := d.am.PrepareArtifact(ctx, EtcdArtifactType, d.config.Etcd.Artifact); err != nil {
		return err
	}

	bin, err := d.am.BinaryPath(EtcdArtifactType, d.config.Etcd.Artifact)
	if err != nil {
		return err
	}

	if err = d.bm.Etcd.Start(ctx, bin); err != nil {
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
	return fmt.Errorf("etcd is not ready in 10 second! You can find its logs in %s", path.Join(d.logsDir, "etcd"))
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

	d.logger.V(3).Info("Cluster shutting down. Cleaning allocated resources.")

	<-ctx.Done()
	// Delete cluster after closing, which can only happens in the foreground.
	if err := d.deleteGreptimeDBClusterForeground(ctx); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) Config() *config.Config {
	return d.config
}
