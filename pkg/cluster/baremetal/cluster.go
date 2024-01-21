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

package baremetal

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/components"
	"github.com/GreptimeTeam/gtctl/pkg/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
)

type Cluster struct {
	config       *config.BareMetalClusterConfig
	createNoDirs bool
	enableCache  bool

	am artifacts.Manager
	mm metadata.Manager
	cc *ClusterComponents

	logger logger.Logger
	stop   context.CancelFunc
	ctx    context.Context
	wg     sync.WaitGroup
}

// ClusterComponents describes all the components need to be deployed under bare-metal mode.
type ClusterComponents struct {
	MetaSrv  components.ClusterComponent
	Datanode components.ClusterComponent
	Frontend components.ClusterComponent
	Etcd     components.ClusterComponent
}

func NewClusterComponents(config *config.BareMetalClusterComponentsConfig, workingDirs components.WorkingDirs,
	wg *sync.WaitGroup, logger logger.Logger) *ClusterComponents {
	return &ClusterComponents{
		MetaSrv:  components.NewMetaSrv(config.MetaSrv, workingDirs, wg, logger),
		Datanode: components.NewDataNode(config.Datanode, config.MetaSrv.ServerAddr, workingDirs, wg, logger),
		Frontend: components.NewFrontend(config.Frontend, config.MetaSrv.ServerAddr, workingDirs, wg, logger),
		Etcd:     components.NewEtcd(workingDirs, wg, logger),
	}
}

type Option func(cluster *Cluster)

// WithReplaceConfig replaces current cluster config with given config.
func WithReplaceConfig(cfg *config.BareMetalClusterConfig) Option {
	return func(c *Cluster) {
		c.config = cfg
	}
}

func WithGreptimeVersion(version string) Option {
	return func(c *Cluster) {
		c.config.Cluster.Artifact.Version = version
	}
}

func WithEnableCache(enableCache bool) Option {
	return func(c *Cluster) {
		c.enableCache = enableCache
	}
}

func WithCreateNoDirs() Option {
	return func(c *Cluster) {
		c.createNoDirs = true
	}
}

func NewCluster(l logger.Logger, clusterName string, opts ...Option) (cluster.Operations, error) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	c := &Cluster{
		logger: l,
		config: config.DefaultBareMetalConfig(),
		ctx:    ctx,
		stop:   stop,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if err := config.ValidateConfig(c.config); err != nil {
		return nil, err
	}

	// Configure Metadata Manager
	mm, err := metadata.New("")
	if err != nil {
		return nil, err
	}
	c.mm = mm

	// Configure Artifact Manager.
	am, err := artifacts.NewManager(l)
	if err != nil {
		return nil, err
	}
	c.am = am

	// Configure Cluster Components.
	mm.AllocateClusterScopeDirs(clusterName)
	if !c.createNoDirs {
		if err = mm.CreateClusterScopeDirs(c.config); err != nil {
			return nil, err
		}
	}
	csd := mm.GetClusterScopeDirs()
	c.cc = NewClusterComponents(c.config.Cluster, components.WorkingDirs{
		DataDir: csd.DataDir,
		LogsDir: csd.LogsDir,
		PidsDir: csd.PidsDir,
	}, &c.wg, c.logger)

	return c, nil
}
