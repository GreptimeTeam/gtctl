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

package delete

import (
	"context"
	"fmt"
	"strings"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type deleteClusterOptions struct {
	Namespace    string
	TearDownEtcd bool
}

func NewDeleteClusterCommand(l logger.Logger) *cobra.Command {
	var options deleteClusterOptions

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GreptimeDB cluster",
		Long:  `Delete a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			clusterName, namespace := args[0], options.Namespace
			l.V(0).Infof("Deleting cluster '%s' in namespace '%s'...\n", logger.Bold(clusterName), logger.Bold(namespace))

			k8sDeployer, err := k8s.NewDeployer(l)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			name := types.NamespacedName{Namespace: options.Namespace, Name: clusterName}.String()
			cluster, err := k8sDeployer.GetGreptimeDBCluster(ctx, name, nil)
			if errors.IsNotFound(err) {
				l.V(0).Infof("Cluster '%s' in '%s' not found\n", clusterName, namespace)
				return nil
			}
			if err != nil {
				return err
			}

			rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
			if !ok {
				return fmt.Errorf("invalid cluster type")
			}

			name = types.NamespacedName{Namespace: options.Namespace, Name: clusterName}.String()
			if err := k8sDeployer.DeleteGreptimeDBCluster(ctx, name, nil); err != nil {
				return err
			}

			// TODO(zyy17): Should we wait until the cluster is actually deleted?
			l.V(0).Infof("Cluster '%s' in namespace '%s' is deleted!\n", clusterName, namespace)

			if options.TearDownEtcd {
				etcdNamespace := strings.Split(strings.Split(rawCluster.Spec.Meta.EtcdEndpoints[0], ".")[1], ":")[0]
				l.V(0).Infof("Deleting etcd cluster in namespace '%s'...\n", logger.Bold(etcdNamespace))
				name = types.NamespacedName{Namespace: etcdNamespace, Name: EtcdClusterName(clusterName)}.String()
				if err := k8sDeployer.DeleteEtcdCluster(ctx, name, nil); err != nil {
					return err
				}
				l.V(0).Infof("Etcd cluster in namespace '%s' is deleted!\n", etcdNamespace)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")

	return cmd
}

func EtcdClusterName(clusterName string) string {
	return fmt.Sprintf("%s-etcd", clusterName)
}
