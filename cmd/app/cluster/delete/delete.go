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

package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type deleteClusterCliOptions struct {
	Namespace    string
	TearDownEtcd bool
}

func NewDeleteClusterCommand(l log.Logger) *cobra.Command {
	var options deleteClusterCliOptions

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GreptimeDB cluster",
		Long:  `Delete a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			clusterName, namespace := args[0], options.Namespace
			l.Infof("⚠️ Deleting cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(namespace))

			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			_, err = m.GetCluster(ctx, &manager.GetClusterOptions{
				ClusterName: clusterName,
				Namespace:   options.Namespace,
			})
			if err != nil && errors.IsNotFound(err) {
				l.Infof("Cluster '%s' in '%s' not found\n", clusterName, namespace)
				return nil
			} else if err != nil {
				return err
			}

			if err := m.DeleteCluster(ctx, &manager.DeleteClusterOption{
				ClusterName:  clusterName,
				Namespace:    options.Namespace,
				TearDownEtcd: options.TearDownEtcd,
			}); err != nil {
				return err
			}

			// TODO(zyy17): Should we wait until the cluster is actually deleted?
			l.Infof("Cluster '%s' in namespace '%s' is deleted!\n", clusterName, namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")

	return cmd
}
