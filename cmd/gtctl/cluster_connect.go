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
	"github.com/GreptimeTeam/gtctl/pkg/cluster/kubernetes"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type clusterConnectCliOptions struct {
	Namespace string
	Protocol  string
}

func NewConnectCommand(l logger.Logger) *cobra.Command {
	var options clusterConnectCliOptions

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a GreptimeDB cluster",
		Long:  `Connect to a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
				protocol    opt.ConnectProtocol
			)

			cluster, err := kubernetes.NewCluster(l)
			if err != nil {
				return err
			}

			switch options.Protocol {
			case "mysql":
				protocol = opt.MySQL
			case "pg", "psql", "postgres":
				protocol = opt.Postgres
			default:
				return fmt.Errorf("unsupported connection protocol: %s", options.Protocol)
			}
			connectOptions := &opt.ConnectOptions{
				Namespace: options.Namespace,
				Name:      clusterName,
				Protocol:  protocol,
			}

			return cluster.Connect(ctx, connectOptions)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().StringVarP(&options.Protocol, "protocol", "p", "mysql", "Specify a database protocol, like mysql or pg.")

	return cmd
}
