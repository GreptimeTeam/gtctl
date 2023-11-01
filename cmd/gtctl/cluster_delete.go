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

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/baremetal"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/kubernetes"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type clusterDeleteOptions struct {
	Namespace    string
	TearDownEtcd bool

	// The options for deleting GreptimeDB cluster in bare-metal.
	BareMetal bool
}

func NewDeleteClusterCommand(l logger.Logger) *cobra.Command {
	var options clusterDeleteOptions

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GreptimeDB cluster",
		Long:  `Delete a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			clusterName := args[0]
			var (
				cluster opt.Operations
				err     error
				ctx     = context.TODO()
			)

			if options.BareMetal {
				cluster, err = baremetal.NewCluster(l, clusterName, baremetal.WithCreateNoDirs())
			} else {
				cluster, err = kubernetes.NewCluster(l)
			}
			if err != nil {
				return err
			}

			deleteOptions := &opt.DeleteOptions{
				Namespace: options.Namespace,
				Name:      clusterName,
			}
			return cluster.Delete(ctx, deleteOptions)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")
	cmd.Flags().BoolVar(&options.BareMetal, "bare-metal", false, "Get the greptimedb cluster on bare-metal environment.")

	return cmd
}
