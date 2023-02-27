// Copyright 2022 Greptime Team
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

package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

func NewListClustersCommand(l logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all GreptimeDB clusters",
		Long:  `List all GreptimeDB clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sDeployer, err := k8s.NewDeployer(l)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			clusters, err := k8sDeployer.ListGreptimeDBClusters(ctx, nil)
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			if errors.IsNotFound(err) || (clusters != nil && len(clusters) == 0) {
				l.Error("clusters not found\n")
				return nil
			}

			// TODO(zyy17): more human friendly output format.
			for _, cluster := range clusters {
				rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
				if !ok {
					return fmt.Errorf("invalid cluster type")
				}
				l.V(0).Infof("Cluster '%s' in '%s' namespace is running, create at %s\n",
					rawCluster.Name, rawCluster.Namespace, rawCluster.CreationTimestamp)
			}

			return nil
		},
	}

	return cmd
}
