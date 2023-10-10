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
	"context"
	"fmt"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/api/query"
)

var _ query.Getter = &Cluster{}

func (c *Cluster) Get(ctx context.Context, options *query.Options) error {
	cluster, err := c.get(ctx, options)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) || cluster == nil {
		return fmt.Errorf("cluster not found")
	}

	c.logger.V(0).Infof("Cluster '%s' in '%s' namespace is running, create at %s\n",
		options.Name, options.Namespace, cluster.CreationTimestamp)

	return nil
}

func (c *Cluster) get(ctx context.Context, options *query.Options) (*greptimedbclusterv1alpha1.GreptimeDBCluster, error) {
	cluster, err := c.client.GetCluster(ctx, options.Name, options.Namespace)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
