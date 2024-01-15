// Copyright 2024 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
)

func (c *Cluster) Scale(ctx context.Context, options *opt.ScaleOptions) error {
	cluster, err := c.get(ctx, &opt.GetOptions{
		Namespace: options.Namespace,
		Name:      options.Name,
	})
	if err != nil {
		return err
	}

	c.scale(options, cluster)
	c.logger.V(0).Infof("Scaling cluster %s in %s from %d to %d\n",
		options.Name, options.Namespace, options.OldReplicas, options.NewReplicas)

	if err = c.client.UpdateCluster(ctx, options.Namespace, cluster); err != nil {
		return err
	}

	return c.client.WaitForClusterReady(ctx, options.Name, options.Namespace, c.timeout)
}

func (c *Cluster) scale(options *opt.ScaleOptions, cluster *greptimedbclusterv1alpha1.GreptimeDBCluster) {
	switch options.ComponentType {
	case greptimedbclusterv1alpha1.FrontendComponentKind:
		options.OldReplicas = cluster.Spec.Frontend.Replicas
		cluster.Spec.Frontend.Replicas = options.NewReplicas
	case greptimedbclusterv1alpha1.DatanodeComponentKind:
		options.OldReplicas = cluster.Spec.Datanode.Replicas
		cluster.Spec.Datanode.Replicas = options.NewReplicas
	case greptimedbclusterv1alpha1.MetaComponentKind:
		options.OldReplicas = cluster.Spec.Meta.Replicas
		cluster.Spec.Meta.Replicas = options.NewReplicas
	}
}
