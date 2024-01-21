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

package kubernetes

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
)

func (c *Cluster) Delete(ctx context.Context, options *opt.DeleteOptions) error {
	cluster, err := c.get(ctx, &opt.GetOptions{
		Namespace: options.Namespace,
		Name:      options.Name,
	})
	if errors.IsNotFound(err) || cluster == nil {
		c.logger.V(0).Infof("Cluster '%s' in '%s' not found", options.Name, options.Namespace)
		return nil
	}
	if err != nil {
		return err
	}

	// TODO: should wait cluster to be terminated?
	c.logger.V(0).Infof("Deleting cluster '%s' in namespace '%s'...", options.Name, options.Namespace)
	if err = c.deleteCluster(ctx, options); err != nil {
		return err
	}
	c.logger.V(0).Infof("Cluster '%s' in namespace '%s' is deleted!", options.Name, options.Namespace)

	if options.TearDownEtcd {
		c.logger.V(0).Infof("Deleting etcd cluster in namespace '%s'...", options.Namespace)
		if err = c.deleteEtcdCluster(ctx, &opt.DeleteOptions{
			Namespace: options.Namespace,
			Name:      EtcdClusterName(options.Name),
		}); err != nil {
			return err
		}
		c.logger.V(0).Infof("Etcd cluster in namespace '%s' is deleted!", options.Namespace)
	}
	return nil
}

func (c *Cluster) deleteCluster(ctx context.Context, options *opt.DeleteOptions) error {
	return c.client.DeleteCluster(ctx, options.Name, options.Namespace)
}

func (c *Cluster) deleteEtcdCluster(ctx context.Context, options *opt.DeleteOptions) error {
	return c.client.DeleteEtcdCluster(ctx, options.Name, options.Namespace)
}
