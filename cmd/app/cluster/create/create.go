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

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type createClusterCliOptions struct {
	Namespace            string
	OperatorNamespace    string
	EtcdNamespace        string
	StorageClassName     string
	StorageSize          string
	StorageRetainPolicy  string
	GreptimeDBVersion    string
	OperatorVersion      string
	Registry             string
	EtcdEndpoint         string
	EtcdChartsVersion    string
	EtcdStorageClassName string
	EtcdStorageSize      string

	DryRun  bool
	Timeout int
}

func NewCreateClusterCommand(l log.Logger) *cobra.Command {
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
				Namespace:       options.OperatorNamespace,
				Timeout:         time.Duration(options.Timeout) * time.Second,
				DryRun:          options.DryRun,
				OperatorVersion: options.OperatorVersion,
				Registry:        options.Registry,
			}

			var (
				clusterName = args[0]

				// TODO(zyy17): should use timeout context.
				ctx = context.TODO()
			)

			l.Infof("☕️ Creating cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(options.Namespace))
			l.Infof("☕️ Start to create greptimedb-operator...\n")
			if err := log.StartSpinning("Creating greptimedb-operator...", func() error {
				return m.CreateOperator(ctx, createOperatorOptions)
			}); err != nil {
				return err
			}
			l.Infof("🎉 Finish to create greptimedb-operator.\n")

			l.Infof("☕️ Start to create etcd cluster...\n")
			createEtcdOptions := &manager.CreateEtcdOptions{
				Name:                 args[0] + "-" + "etcd",
				Namespace:            options.EtcdNamespace,
				Timeout:              time.Duration(options.Timeout) * time.Second,
				DryRun:               options.DryRun,
				Registry:             options.Registry,
				EtcdChartsVersion:    options.EtcdChartsVersion,
				EtcdStorageClassName: options.EtcdStorageClassName,
				EtcdStorageSize:      options.EtcdStorageSize,
			}
			if err := log.StartSpinning("Creating etcd cluster...", func() error {
				return m.CreateEtcdCluster(ctx, createEtcdOptions)
			}); err != nil {
				return err
			}
			l.Infof("🎉 Finish to create etcd cluster.\n")

			l.Infof("☕️ Start to create GreptimeDB cluster...\n")
			createClusterOptions := &manager.CreateClusterOptions{
				ClusterName:         args[0],
				Namespace:           options.Namespace,
				StorageClassName:    options.StorageClassName,
				StorageSize:         options.StorageSize,
				StorageRetainPolicy: options.StorageRetainPolicy,
				Timeout:             time.Duration(options.Timeout) * time.Second,
				DryRun:              options.DryRun,
				GreptimeDBVersion:   options.GreptimeDBVersion,
				Registry:            options.Registry,
				EtcdEndPoint:        options.EtcdEndpoint,
			}

			if err := log.StartSpinning("Creating GreptimeDB cluster", func() error {
				return m.CreateCluster(ctx, createClusterOptions)
			}); err != nil {
				return err
			}

			if !options.DryRun {
				l.Infof("🎉 Finish to create GreptimeDB cluster '%s'.\n", log.Bold(clusterName))
				l.Infof("💡 You can use `%s` to access the database.\n", log.Bold(fmt.Sprintf("kubectl port-forward svc/%s-frontend -n %s 4002:4002", clusterName, options.Namespace)))
				l.Infof("😊 Thank you for using %s!\n", log.Bold("GreptimeDB"))
				l.Infof("🔑 %s\n", log.Bold("Invest in Data, Harvest over Time."))
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
	cmd.Flags().StringVar(&options.GreptimeDBVersion, "version", manager.DefaultGreptimeDBChartVersion, "The GreptimeDB version.")
	cmd.Flags().StringVar(&options.OperatorVersion, "operator-version", manager.DefaultGreptimeDBOperatorChartVersion, "The greptimedb-operator version.")
	cmd.Flags().StringVar(&options.Registry, "registry", "", "The image registry")
	cmd.Flags().StringVar(&options.EtcdEndpoint, "etcd-endpoint", "mydb-etcd-svc.default:2379", "The etcd end point")
	cmd.Flags().StringVar(&options.EtcdChartsVersion, "etcd-chart-version", manager.DefaultEtcdChartVersion, "The greptimedb-etcd helm chart version")
	cmd.Flags().StringVar(&options.EtcdNamespace, "etcd-namespace", "default", "The namespace of etcd cluster.")
	cmd.Flags().StringVar(&options.EtcdStorageClassName, "etcd-storage-class-name", "standard", "The etcd storage class name.")
	cmd.Flags().StringVar(&options.EtcdStorageSize, "etcd-storage-size", "10Gi", "the etcd persistent volume size.")

	return cmd
}
