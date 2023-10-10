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

package kubernetes

import (
	"time"

	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type Cluster struct {
	helmManager *helm.Manager
	client      *kube.Client
	logger      logger.Logger

	timeout time.Duration
	dryRun  bool
}

type Option func(cluster *Cluster)

func NewCluster(l logger.Logger, opts ...Option) (*Cluster, error) {
	hm, err := helm.NewManager(l)
	if err != nil {
		return nil, err
	}

	cluster := &Cluster{
		helmManager: hm,
		logger:      l,
	}
	for _, opt := range opts {
		opt(cluster)
	}

	var client *kube.Client
	if !cluster.dryRun {
		client, err = kube.NewClient("")
		if err != nil {
			return nil, err
		}
	}
	cluster.client = client

	return cluster, nil
}
