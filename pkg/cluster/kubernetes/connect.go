// Copyright 2023 Greptime Team
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
	"fmt"
	"strconv"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/connector"
)

func (c *Cluster) Connect(ctx context.Context, options *opt.ConnectOptions) error {
	cluster, err := c.get(ctx, &opt.GetOptions{
		Namespace: options.Namespace,
		Name:      options.Name,
	})
	if err != nil && errors.IsNotFound(err) {
		c.logger.V(0).Infof("cluster %s in %s not found", options.Name, options.Namespace)
		return nil
	}

	switch options.Protocol {
	case opt.MySQL:
		if err = c.connectMySQL(cluster); err != nil {
			return fmt.Errorf("error connecting to mysql: %v", err)
		}
	case opt.Postgres:
		if err = c.connectPostgres(cluster); err != nil {
			return fmt.Errorf("error connecting to postgres: %v", err)
		}
	default:
		return fmt.Errorf("unsupported connect protocol type")
	}

	return nil
}

func (c *Cluster) connectMySQL(cluster *greptimedbclusterv1alpha1.GreptimeDBCluster) error {
	return connector.Mysql(strconv.Itoa(int(cluster.Spec.MySQLServicePort)), cluster.Name, c.logger)
}

func (c *Cluster) connectPostgres(cluster *greptimedbclusterv1alpha1.GreptimeDBCluster) error {
	return connector.PostgresSQL(strconv.Itoa(int(cluster.Spec.PostgresServicePort)), cluster.Name, c.logger)
}
