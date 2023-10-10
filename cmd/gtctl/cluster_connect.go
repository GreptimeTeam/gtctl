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

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/GreptimeTeam/gtctl/pkg/connector"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type clusterConnectCliOptions struct {
	Namespace string
	Protocol  string
}

// TODO(sh2): should be abstracted into one api interface `connect`
func NewConnectCommand(l logger.Logger) *cobra.Command {
	const (
		connectionProtocolMySQL    = "mysql"
		connectionProtocolPostgres = "pg"
	)

	var options clusterConnectCliOptions

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a GreptimeDB cluster",
		Long:  `Connect to a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			k8sDeployer, err := k8s.NewDeployer(l)
			if err != nil {
				return err
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
				namespace   = options.Namespace
			)

			name := types.NamespacedName{
				Namespace: options.Namespace,
				Name:      clusterName,
			}.String()
			cluster, err := k8sDeployer.GetGreptimeDBCluster(ctx, name, nil)
			if err != nil && errors.IsNotFound(err) {
				l.Errorf("cluster %s in %s not found\n", clusterName, namespace)
				return nil
			}

			rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
			if !ok {
				return fmt.Errorf("invalid cluster type")
			}

			switch options.Protocol {
			case connectionProtocolMySQL:
				if err = connector.MySQLConnectCommand(rawCluster, l); err != nil {
					return fmt.Errorf("error connecting to mysql: %v", err)
				}
			case connectionProtocolPostgres:
				if err = connector.PostgresSQLConnectCommand(rawCluster, l); err != nil {
					return fmt.Errorf("error connecting to postgres: %v", err)
				}
			default:
				return fmt.Errorf("database type not supported: %s", options.Protocol)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().StringVarP(&options.Protocol, "protocol", "p", "mysql", "Specify a database, like mysql or pg.")

	return cmd
}
