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

package create

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
	"github.com/GreptimeTeam/gtctl/pkg/status"
)

type createClusterCliOptions struct {
	Namespace                      string
	OperatorNamespace              string
	EtcdNamespace                  string
	StorageClassName               string
	StorageSize                    string
	StorageRetainPolicy            string
	GreptimeDBChartVersion         string
	GreptimeDBOperatorChartVersion string
	ImageRegistry                  string
	EtcdEndpoint                   string
	EtcdChartVersion               string
	EtcdStorageClassName           string
	EtcdStorageSize                string

	DryRun  bool
	Timeout int
}

func NewCreateClusterCommand(l logger.Logger) *cobra.Command {
	var options createClusterCliOptions

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a GreptimeDB cluster",
		Long:  `Create a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			m, err := manager.New(l, options.DryRun)
			if err != nil {
				return err
			}

			createOperatorOptions := &manager.CreateOperatorOptions{
				Namespace:              options.OperatorNamespace,
				Timeout:                time.Duration(options.Timeout) * time.Second,
				DryRun:                 options.DryRun,
				GreptimeDBChartVersion: options.GreptimeDBOperatorChartVersion,
				ImageRegistry:          options.ImageRegistry,
			}

			var (
				clusterName = args[0]
				etcdSvcName = fmt.Sprintf("%s-etcd-svc", args[0])

				// TODO(zyy17): should use timeout context.
				ctx = context.TODO()
			)

			spinner, err := status.NewSpinner()
			if err != nil {
				return err
			}

			l.V(0).Infof("Creating GreptimeDB cluster '%s' in namespace '%s' ...", logger.Bold(clusterName), logger.Bold(options.Namespace))

			spinner.Start("Installing greptimedb-operator...")
			if err := m.CreateOperator(ctx, createOperatorOptions); err != nil {
				spinner.Stop(false, "Installing greptimedb-operator failed")
				return err
			}
			spinner.Stop(true, "Installing greptimedb-operator successfully 🎉")

			spinner.Start("Installing etcd cluster...")
			createEtcdOptions := &manager.CreateEtcdOptions{
				Name:                 args[0] + "-etcd",
				Namespace:            options.EtcdNamespace,
				Timeout:              time.Duration(options.Timeout) * time.Second,
				DryRun:               options.DryRun,
				ImageRegistry:        options.ImageRegistry,
				EtcdChartVersion:     options.EtcdChartVersion,
				EtcdStorageClassName: options.EtcdStorageClassName,
				EtcdStorageSize:      options.EtcdStorageSize,
			}
			if err := m.CreateEtcdCluster(ctx, createEtcdOptions); err != nil {
				spinner.Stop(false, "Installing etcd cluster failed")
				return err
			}
			spinner.Stop(true, "Installing etcd cluster successfully 🎉")

			spinner.Start("Installing GreptimeDB cluster...")
			createClusterOptions := &manager.CreateClusterOptions{
				ClusterName:            args[0],
				Namespace:              options.Namespace,
				StorageClassName:       options.StorageClassName,
				StorageSize:            options.StorageSize,
				StorageRetainPolicy:    options.StorageRetainPolicy,
				Timeout:                time.Duration(options.Timeout) * time.Second,
				DryRun:                 options.DryRun,
				GreptimeDBChartVersion: options.GreptimeDBChartVersion,
				ImageRegistry:          options.ImageRegistry,
				EtcdEndPoint:           fmt.Sprintf("%s.%s:2379", etcdSvcName, options.EtcdNamespace),
			}
			if err := m.CreateCluster(ctx, createClusterOptions); err != nil {
				spinner.Stop(false, "Installing GreptimeDB cluster failed")
				return err
			}
			spinner.Stop(true, "Installing GreptimeDB cluster successfully 🎉")

			if !options.DryRun {
				l.V(0).Infof("\nNow you can use the following commands to access the GreptimeDB cluster:")
				l.V(0).Infof("\n%s", logger.Bold("MySQL >"))
				l.V(0).Infof("%s", fmt.Sprintf("%s kubectl port-forward svc/%s-frontend -n %s 4002:4002 > connections-mysql.out &", logger.Bold("$"), clusterName, options.Namespace))
				l.V(0).Infof("%s", fmt.Sprintf("%s mysql -h 127.0.0.1 -P 4002", logger.Bold("$")))
				l.V(0).Infof("\n%s", logger.Bold("PostgreSQL >"))
				l.V(0).Infof("%s", fmt.Sprintf("%s kubectl port-forward svc/%s-frontend -n %s 4003:4003 > connections-pg.out &", logger.Bold("$"), clusterName, options.Namespace))
				l.V(0).Infof("%s", fmt.Sprintf("%s psql -h 127.0.0.1 -p 4003", logger.Bold("$")))
				l.V(0).Infof("\nThank you for using %s! Check for more information on %s. 😊", logger.Bold("GreptimeDB"), logger.Bold("https://greptime.com"))
				l.V(0).Infof("\n%s 🔑", logger.Bold("Invest in Data, Harvest over Time."))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&options.OperatorNamespace, "operator-namespace", "default", "The namespace of deploying greptimedb-operator.")
	cmd.Flags().StringVar(&options.StorageClassName, "storage-class-name", "standard", "Datanode storage class name.")
	cmd.Flags().StringVar(&options.StorageSize, "storage-size", "10Gi", "Datanode persistent volume size.")
	cmd.Flags().StringVar(&options.StorageRetainPolicy, "retain-policy", "Retain", "Datanode pvc retain policy.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Output the manifests without applying them.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")
	cmd.Flags().StringVar(&options.GreptimeDBChartVersion, "greptimedb-chart-version", "", "The greptimedb helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.GreptimeDBOperatorChartVersion, "greptimedb-operator-chart-version", "", "The greptimedb-operator helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.EtcdChartVersion, "etcd-chart-version", "", "The greptimedb-etcd helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.ImageRegistry, "image-registry", "", "The image registry")
	cmd.Flags().StringVar(&options.EtcdNamespace, "etcd-namespace", "default", "The namespace of etcd cluster.")
	cmd.Flags().StringVar(&options.EtcdStorageClassName, "etcd-storage-class-name", "standard", "The etcd storage class name.")
	cmd.Flags().StringVar(&options.EtcdStorageSize, "etcd-storage-size", "10Gi", "the etcd persistent volume size.")

	return cmd
}
