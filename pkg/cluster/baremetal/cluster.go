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
	"os/signal"
	"sync"
	"syscall"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	"github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/metadata"
)

type Cluster struct {
	logger logger.Logger
	config *Config
	wg     sync.WaitGroup
	ctx    context.Context

	createNoDirs bool
	enableCache  bool

	am artifacts.Manager
	mm metadata.Manager
	bm *component.BareMetalCluster
}

type Option func(cluster *Cluster)

func NewCluster(l logger.Logger, clusterName string, opts ...Option) (cluster.Operations, error) {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	c := &Cluster{
		logger: l,
		config: DefaultConfig(),
		ctx:    ctx,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if err := ValidateConfig(c.config); err != nil {
		return nil, err
	}

	// Configure Metadata Manager
	mm, err := metadata.New("", clusterName)
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

	// TODO(sh2): implement it in the following PR
	// if !cluster.createNoDirs {}

	return c, nil
}
