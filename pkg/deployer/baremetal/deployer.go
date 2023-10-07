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
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type Deployer struct {
	logger logger.Logger
	config *config.Config
	am     *ArtifactManager
	wg     sync.WaitGroup
	bm     *component.BareMetalCluster
	ctx    context.Context

	createNoDirs      bool
	workingDirs       component.WorkingDirs
	clusterDir        string
	baseDir           string
	clusterConfigPath string

	alwaysDownload bool
}

var _ Interface = &Deployer{}

type Option func(*Deployer)

func NewDeployer(l logger.Logger, clusterName string, opts ...Option) (Interface, error) {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	d := &Deployer{
		logger: l,
		config: config.DefaultConfig(),
		ctx:    ctx,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	if err := ValidateConfig(d.config); err != nil {
		return nil, err
	}

	if len(d.baseDir) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		d.baseDir = path.Join(homeDir, config.GtctlDir)
	}

	if err := fileutils.EnsureDir(d.baseDir); err != nil {
		return nil, err
	}

	am, err := NewArtifactManager(d.baseDir, l, d.alwaysDownload)
	if err != nil {
		return nil, err
	}
	d.am = am

	d.initClusterDirsAndPath(clusterName)

	if !d.createNoDirs {
		if err = d.createClusterDirs(); err != nil {
			return nil, err
		}

		d.bm = component.NewBareMetalCluster(d.config.Cluster, d.workingDirs, &d.wg, d.logger)

		// Save a copy of cluster config in yaml format.
		if err = d.createClusterConfigFile(); err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (d *Deployer) initClusterDirsAndPath(clusterName string) {
	// Dirs
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}
		clusterDir = path.Join(d.baseDir, clusterName)

		// ${HOME}/${GtctlDir}/${ClusterName}/logs
		logsDir = path.Join(clusterDir, config.LogsDir)

		// ${HOME}/${GtctlDir}/${ClusterName}/data
		dataDir = path.Join(clusterDir, config.DataDir)

		// ${HOME}/${GtctlDir}/${ClusterName}/pids
		pidsDir = path.Join(clusterDir, config.PidsDir)
	)

	// Path
	var (
		// ${HOME}/${GtctlDir}/${ClusterName}/${ClusterName}.yaml
		clusterConfigPath = path.Join(clusterDir, fmt.Sprintf("%s.yaml", clusterName))
	)

	d.clusterDir = clusterDir
	d.workingDirs = component.WorkingDirs{
		LogsDir: logsDir,
		DataDir: dataDir,
		PidsDir: pidsDir,
	}
	d.clusterConfigPath = clusterConfigPath
}

func (d *Deployer) createClusterDirs() error {
	dirs := []string{
		d.clusterDir,
		d.workingDirs.LogsDir,
		d.workingDirs.DataDir,
		d.workingDirs.PidsDir,
	}

	for _, dir := range dirs {
		if err := fileutils.EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deployer) createClusterConfigFile() error {
	f, err := os.Create(d.clusterConfigPath)
	if err != nil {
		return err
	}

	metaConfig := config.MetaConfig{
		Config:        d.config,
		CreationDate:  time.Now(),
		ClusterDir:    d.clusterDir,
		ForegroundPid: os.Getpid(),
	}

	out, err := yaml.Marshal(metaConfig)
	if err != nil {
		return err
	}

	if _, err = f.Write(out); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	return nil
}

// WithMergeConfig merges config with current deployer config.
// It will perform WithReplaceConfig if any error occurs during merging or receive nil raw config.
func WithMergeConfig(cfg *config.Config, rawConfig []byte) Option {
	if len(rawConfig) == 0 {
		return WithReplaceConfig(cfg)
	}

	return func(d *Deployer) {
		defaultConfig, err := yaml.Marshal(d.config)
		if err != nil {
			d.config = cfg
			return
		}

		out, err := fileutils.MergeYAML(defaultConfig, rawConfig)
		if err != nil {
			d.config = cfg
			return
		}

		var newConfig config.Config
		if err = yaml.Unmarshal(out, &newConfig); err != nil {
			d.config = cfg
			return
		}

		d.config = &newConfig
	}
}

// WithReplaceConfig replaces config with current deployer config.
func WithReplaceConfig(cfg *config.Config) Option {
	return func(d *Deployer) {
		d.config = cfg
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

func WithCreateNoDirs() Option {
	return func(d *Deployer) {
		d.createNoDirs = true
	}
}

func WithBaseDir(baseDir string) Option {
	return func(d *Deployer) {
		d.baseDir = baseDir
	}
}

func (d *Deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	_, err := os.Stat(d.clusterDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cluster %s is not exist", name)
	}
	if err != nil {
		return nil, err
	}

	ok, err := fileutils.IsFileExists(d.clusterConfigPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("cluster %s is not exist", name)
	}

	var cluster config.MetaConfig
	in, err := os.ReadFile(d.clusterConfigPath)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(in, &cluster); err != nil {
		return nil, err
	}

	return &GreptimeDBCluster{
		Raw: &cluster,
	}, nil
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

	if err := d.bm.MetaSrv.Start(d.ctx, binary); err != nil {
		return err
	}

	if err := d.bm.Datanode.Start(d.ctx, binary); err != nil {
		return err
	}

	if err := d.bm.Frontend.Start(d.ctx, binary); err != nil {
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
func (d *Deployer) deleteGreptimeDBClusterForeground(ctx context.Context, option component.DeleteOptions) error {
	// No matter what options are, the config file of one cluster must be deleted.
	if err := os.Remove(d.clusterConfigPath); err != nil {
		return err
	}

	if option.RetainLogs {
		// It is unnecessary to delete each component resources in cluster since it only retains the logs.
		// So deleting the whole cluster resources excluding logs here would be fine.
		if err := fileutils.DeleteDirIfExists(d.workingDirs.DataDir); err != nil {
			return err
		}
		if err := fileutils.DeleteDirIfExists(d.workingDirs.PidsDir); err != nil {
			return err
		}
	} else {
		// It is unnecessary to delete each component resources in cluster since it has nothing to retain.
		// So deleting the whole cluster resources here would be fine.
		if err := fileutils.DeleteDirIfExists(d.clusterDir); err != nil {
			return err
		}
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

	if err = d.bm.Etcd.Start(d.ctx, bin); err != nil {
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
	exists, err := fileutils.IsFileExists(etcdctlBin)
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
	return fmt.Errorf("etcd is not ready in 10 second! You can find its logs in %s", path.Join(d.workingDirs.LogsDir, "etcd"))
}

func (d *Deployer) DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *Deployer) CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error {
	// We don't need to implement this method because we don't need to deploy GreptimeDB Operator.
	return fmt.Errorf("only support for k8s Deployer")
}

func (d *Deployer) Wait(ctx context.Context, option component.DeleteOptions) error {
	d.wg.Wait()

	d.logger.V(3).Info("Cluster shutting down. Cleaning allocated resources.")

	<-d.ctx.Done()
	// Delete cluster after closing, which can only happens in the foreground.
	if err := d.deleteGreptimeDBClusterForeground(ctx, option); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) Config() *config.Config {
	return d.config
}
