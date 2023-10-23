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

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

type Deployer struct {
	logger logger.Logger
	config *config.Config
	wg     sync.WaitGroup
	ctx    context.Context

	createNoDirs bool
	enableCache  bool

	am artifacts.Manager
	mm metadata.Manager
	bm *component.BareMetalCluster
}

var _ Interface = &Deployer{}

type Option func(*Deployer)

// TODO(sh2): remove this deployer later
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

	mm, err := metadata.New("", clusterName)
	if err != nil {
		return nil, err
	}
	d.mm = mm

	am, err := artifacts.NewManager(l)
	if err != nil {
		return nil, err
	}
	d.am = am

	if !d.createNoDirs {
		if err = mm.AllocateClusterScopeDirs(); err != nil {
			return nil, err
		}

		csd := mm.GetClusterScopeDir()
		d.bm = component.NewBareMetalCluster(d.config.Cluster, component.WorkingDirs{
			DataDir: csd.DataDir,
			LogsDir: csd.LogsDir,
			PidsDir: csd.PidsDir,
		}, &d.wg, d.logger)

		// Save a copy of cluster config in yaml format.
		if err = mm.AllocateClusterConfigPath(d.config); err != nil {
			return nil, err
		}
	}

	return d, nil
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

func WithEnableCache(enableCache bool) Option {
	return func(d *Deployer) {
		d.enableCache = enableCache
	}
}

func WithCreateNoDirs() Option {
	return func(d *Deployer) {
		d.createNoDirs = true
	}
}

func (d *Deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	csd := d.mm.GetClusterScopeDir()
	_, err := os.Stat(csd.BaseDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cluster %s is not exist", name)
	}
	if err != nil {
		return nil, err
	}

	ok, err := fileutils.IsFileExists(csd.ConfigPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("cluster %s is not exist", name)
	}

	var cluster config.RuntimeConfig
	in, err := os.ReadFile(csd.ConfigPath)
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
	var binPath string
	if d.config.Cluster.Artifact != nil {
		if d.config.Cluster.Artifact.Local != "" {
			binPath = d.config.Cluster.Artifact.Local
		} else {
			src, err := d.am.NewSource(artifacts.GreptimeBinName, d.config.Cluster.Artifact.Version, artifacts.ArtifactTypeBinary, false)
			if err != nil {
				return err
			}

			destDir, err := d.mm.AllocateArtifactFilePath(src, false)
			if err != nil {
				return err
			}

			installDir, err := d.mm.AllocateArtifactFilePath(src, true)
			if err != nil {
				return err
			}

			artifactFile, err := d.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{EnableCache: d.enableCache, BinaryInstallDir: installDir})
			if err != nil {
				return err
			}
			binPath = artifactFile
		}
	}

	if err := d.bm.MetaSrv.Start(d.ctx, binPath); err != nil {
		return err
	}

	if err := d.bm.Datanode.Start(d.ctx, binPath); err != nil {
		return err
	}

	if err := d.bm.Frontend.Start(d.ctx, binPath); err != nil {
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
	csd := d.mm.GetClusterScopeDir()
	if err := os.Remove(csd.ConfigPath); err != nil {
		return err
	}

	if option.RetainLogs {
		// It is unnecessary to delete each component resources in cluster since it only retains the logs.
		// So deleting the whole cluster resources excluding logs here would be fine.
		if err := fileutils.DeleteDirIfExists(csd.DataDir); err != nil {
			return err
		}
		if err := fileutils.DeleteDirIfExists(csd.PidsDir); err != nil {
			return err
		}
	} else {
		// It is unnecessary to delete each component resources in cluster since it has nothing to retain.
		// So deleting the whole cluster resources here would be fine.
		if err := fileutils.DeleteDirIfExists(csd.BaseDir); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deployer) CreateEtcdCluster(ctx context.Context, clusterName string, options *CreateEtcdClusterOptions) error {
	var binPath string
	if d.config.Etcd.Artifact != nil {
		if d.config.Etcd.Artifact.Local != "" {
			binPath = d.config.Etcd.Artifact.Local
		} else {
			src, err := d.am.NewSource(artifacts.EtcdBinName, d.config.Etcd.Artifact.Version, artifacts.ArtifactTypeBinary, false)
			if err != nil {
				return err
			}

			destDir, err := d.mm.AllocateArtifactFilePath(src, false)
			if err != nil {
				return err
			}

			installDir, err := d.mm.AllocateArtifactFilePath(src, true)
			if err != nil {
				return err
			}

			artifactFile, err := d.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{EnableCache: d.enableCache, BinaryInstallDir: installDir})
			if err != nil {
				return err
			}
			binPath = artifactFile
		}
	}

	if err := d.bm.Etcd.Start(d.ctx, binPath); err != nil {
		return err
	}

	if err := d.checkEtcdHealth(binPath); err != nil {
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

	csd := d.mm.GetClusterScopeDir()
	return fmt.Errorf("etcd is not ready in 10 second! You can find its logs in %s", path.Join(csd.LogsDir, "etcd"))
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
