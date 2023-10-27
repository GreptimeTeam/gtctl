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
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/baremetal"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/kubernetes"
	"github.com/GreptimeTeam/gtctl/pkg/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/status"
)

const (
	// Various of support config type
	configOperator = "operator"
	configCluster  = "cluster"
	configEtcd     = "etcd"
)

type clusterCreateCliOptions struct {
	// The options for deploying GreptimeDBCluster in K8s.
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
	EtcdClusterSize                string

	// Values files that set in command line.
	GreptimeDBClusterValuesFile  string
	EtcdClusterValuesFile        string
	GreptimeDBOperatorValuesFile string

	// The options for deploying GreptimeDBCluster in bare-metal.
	BareMetal          bool
	Config             string
	GreptimeBinVersion string
	EnableCache        bool

	// Common options.
	Timeout int
	DryRun  bool
	Set     configValues

	// If UseGreptimeCNArtifacts is true, the creation will download the artifacts(charts and binaries) from 'downloads.greptime.cn'.
	// Also, it will use ACR registry for charts images.
	UseGreptimeCNArtifacts bool
}

type configValues struct {
	rawConfig []string

	operatorConfig string
	clusterConfig  string
	etcdConfig     string
}

// parseConfig parse raw config values and classify it to different
// categories of config type by its prefix.
func (c *configValues) parseConfig() error {
	var (
		operatorConfig []string
		clusterConfig  []string
		etcdConfig     []string
	)

	for _, raw := range c.rawConfig {
		if len(raw) == 0 {
			return fmt.Errorf("cannot parse empty config values")
		}

		var configPrefix, configValue string
		values := strings.Split(raw, ",")

		for _, value := range values {
			value = strings.Trim(value, " ")
			cfg := strings.SplitN(value, ".", 2)
			configPrefix = cfg[0]
			if len(cfg) == 2 {
				configValue = cfg[1]
			} else {
				configValue = configPrefix
			}

			switch configPrefix {
			case configOperator:
				operatorConfig = append(operatorConfig, configValue)
			case configCluster:
				clusterConfig = append(clusterConfig, configValue)
			case configEtcd:
				etcdConfig = append(etcdConfig, configValue)
			default:
				clusterConfig = append(clusterConfig, value)
			}
		}
	}

	if len(operatorConfig) > 0 {
		c.operatorConfig = strings.Join(operatorConfig, ",")
	}
	if len(clusterConfig) > 0 {
		c.clusterConfig = strings.Join(clusterConfig, ",")
	}
	if len(etcdConfig) > 0 {
		c.etcdConfig = strings.Join(etcdConfig, ",")
	}

	return nil
}

func NewCreateClusterCommand(l logger.Logger) *cobra.Command {
	var options clusterCreateCliOptions

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a GreptimeDB cluster",
		Long:  `Create a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewCluster(args, &options, l)
		},
	}

	cmd.Flags().StringVar(&options.OperatorNamespace, "operator-namespace", "default", "The namespace of deploying greptimedb-operator.")
	cmd.Flags().StringVar(&options.StorageClassName, "storage-class-name", "null", "Datanode storage class name.")
	cmd.Flags().StringVar(&options.StorageSize, "storage-size", "10Gi", "Datanode persistent volume size.")
	cmd.Flags().StringVar(&options.StorageRetainPolicy, "retain-policy", "Retain", "Datanode pvc retain policy.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Output the manifests without applying them.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", 600, "Timeout in seconds for the command to complete, -1 means no timeout, default is 10 min.")
	cmd.Flags().StringArrayVar(&options.Set.rawConfig, "set", []string{}, "set values on the command line for greptimedb cluster, etcd and operator (can specify multiple or separate values with commas: eg. cluster.key1=val1,etcd.key2=val2).")
	cmd.Flags().StringVar(&options.GreptimeDBChartVersion, "greptimedb-chart-version", "", "The greptimedb helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.GreptimeDBOperatorChartVersion, "greptimedb-operator-chart-version", "", "The greptimedb-operator helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.EtcdChartVersion, "etcd-chart-version", "", "The greptimedb-etcd helm chart version, use latest version if not specified.")
	cmd.Flags().StringVar(&options.ImageRegistry, "image-registry", "", "The image registry.")
	cmd.Flags().StringVar(&options.EtcdNamespace, "etcd-namespace", "default", "The namespace of etcd cluster.")
	cmd.Flags().StringVar(&options.EtcdStorageClassName, "etcd-storage-class-name", "null", "The etcd storage class name.")
	cmd.Flags().StringVar(&options.EtcdStorageSize, "etcd-storage-size", "10Gi", "the etcd persistent volume size.")
	cmd.Flags().StringVar(&options.EtcdClusterSize, "etcd-cluster-size", "1", "the etcd cluster size.")
	cmd.Flags().BoolVar(&options.BareMetal, "bare-metal", false, "Deploy the greptimedb cluster on bare-metal environment.")
	cmd.Flags().StringVar(&options.GreptimeBinVersion, "greptime-bin-version", "", "The version of greptime binary(can be override by config file).")
	cmd.Flags().StringVar(&options.Config, "config", "", "Configuration to deploy the greptimedb cluster on bare-metal environment.")
	cmd.Flags().BoolVar(&options.EnableCache, "enable-cache", true, "If true, enable cache for downloading artifacts(charts and binaries).")
	cmd.Flags().BoolVar(&options.UseGreptimeCNArtifacts, "use-greptime-cn-artifacts", false, "If true, use greptime-cn artifacts(charts and binaries).")
	cmd.Flags().StringVar(&options.GreptimeDBClusterValuesFile, "greptimedb-cluster-values-file", "", "The values file for greptimedb cluster.")
	cmd.Flags().StringVar(&options.EtcdClusterValuesFile, "etcd-cluster-values-file", "", "The values file for etcd cluster.")
	cmd.Flags().StringVar(&options.GreptimeDBOperatorValuesFile, "greptimedb-operator-values-file", "", "The values file for greptimedb operator.")

	return cmd
}

// NewCluster creates a new cluster.
func NewCluster(args []string, options *clusterCreateCliOptions, l logger.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("cluster name should be set")
	}

	var (
		clusterName = args[0]
		ctx         = context.Background()
		cancel      context.CancelFunc
	)

	if options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Second)
		defer cancel()
	}
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	spinner, err := status.NewSpinner()
	if err != nil {
		return err
	}

	// Parse config values that set in command line.
	if err = options.Set.parseConfig(); err != nil {
		return err
	}

	createOptions := &opt.CreateOptions{
		Namespace: options.Namespace,
		Name:      clusterName,
		Etcd: &opt.CreateEtcdOptions{
			ImageRegistry:          options.ImageRegistry,
			EtcdChartVersion:       options.EtcdChartVersion,
			EtcdStorageClassName:   options.EtcdStorageClassName,
			EtcdStorageSize:        options.EtcdStorageSize,
			EtcdClusterSize:        options.EtcdClusterSize,
			ConfigValues:           options.Set.etcdConfig,
			UseGreptimeCNArtifacts: options.UseGreptimeCNArtifacts,
			ValuesFile:             options.EtcdClusterValuesFile,
		},
		Operator: &opt.CreateOperatorOptions{
			GreptimeDBOperatorChartVersion: options.GreptimeDBOperatorChartVersion,
			ImageRegistry:                  options.ImageRegistry,
			ConfigValues:                   options.Set.operatorConfig,
			UseGreptimeCNArtifacts:         options.UseGreptimeCNArtifacts,
			ValuesFile:                     options.GreptimeDBOperatorValuesFile,
		},
		Cluster: &opt.CreateClusterOptions{
			GreptimeDBChartVersion:      options.GreptimeDBChartVersion,
			ImageRegistry:               options.ImageRegistry,
			InitializerImageRegistry:    options.ImageRegistry,
			DatanodeStorageClassName:    options.StorageClassName,
			DatanodeStorageSize:         options.StorageSize,
			DatanodeStorageRetainPolicy: options.StorageRetainPolicy,
			EtcdEndPoints:               fmt.Sprintf("%s.%s:2379", kubernetes.EtcdClusterName(clusterName), options.EtcdNamespace),
			ConfigValues:                options.Set.clusterConfig,
			UseGreptimeCNArtifacts:      options.UseGreptimeCNArtifacts,
			ValuesFile:                  options.GreptimeDBClusterValuesFile,
		},
	}

	var cluster opt.Operations
	if options.BareMetal {
		l.V(0).Infof("Creating GreptimeDB cluster '%s' on bare-metal", logger.Bold(clusterName))

		var opts []baremetal.Option
		opts = append(opts, baremetal.WithEnableCache(options.EnableCache))
		if len(options.GreptimeBinVersion) > 0 {
			opts = append(opts, baremetal.WithGreptimeVersion(options.GreptimeBinVersion))
		}
		if len(options.Config) > 0 {
			var cfg config.BareMetalClusterConfig
			raw, err := os.ReadFile(options.Config)
			if err != nil {
				return err
			}
			if err = yaml.Unmarshal(raw, &cfg); err != nil {
				return err
			}

			opts = append(opts, baremetal.WithReplaceConfig(&cfg))
		}

		cluster, err = baremetal.NewCluster(l, clusterName, opts...)
		if err != nil {
			return err
		}
	} else {
		l.V(0).Infof("Creating GreptimeDB cluster '%s' in namespace '%s'", logger.Bold(clusterName), logger.Bold(options.Namespace))

		cluster, err = kubernetes.NewCluster(l,
			kubernetes.WithDryRun(options.DryRun),
			kubernetes.WithTimeout(time.Duration(options.Timeout)*time.Second))
		if err != nil {
			return err
		}
	}

	if err = cluster.Create(ctx, createOptions, spinner); err != nil {
		return err
	}

	if !options.DryRun {
		printTips(l, clusterName, options)
	}

	if options.BareMetal {
		bm, _ := cluster.(*baremetal.Cluster)
		if err = bm.Wait(ctx, false); err != nil {
			return err
		}
	}

	return nil
}

func printTips(l logger.Logger, clusterName string, options *clusterCreateCliOptions) {
	l.V(0).Infof("\nNow you can use the following commands to access the GreptimeDB cluster:")
	l.V(0).Infof("\n%s", logger.Bold("MySQL >"))
	if !options.BareMetal {
		l.V(0).Infof("%s", fmt.Sprintf("%s kubectl port-forward svc/%s-frontend -n %s 4002:4002 > connections-mysql.out &", logger.Bold("$"), clusterName, options.Namespace))
	}
	l.V(0).Infof("%s", fmt.Sprintf("%s mysql -h 127.0.0.1 -P 4002", logger.Bold("$")))
	l.V(0).Infof("\n%s", logger.Bold("PostgreSQL >"))
	if !options.BareMetal {
		l.V(0).Infof("%s", fmt.Sprintf("%s kubectl port-forward svc/%s-frontend -n %s 4003:4003 > connections-pg.out &", logger.Bold("$"), clusterName, options.Namespace))
	}
	l.V(0).Infof("%s", fmt.Sprintf("%s psql -h 127.0.0.1 -p 4003 -d public", logger.Bold("$")))
	l.V(0).Infof("\nThank you for using %s! Check for more information on %s. 😊", logger.Bold("GreptimeDB"), logger.Bold("https://greptime.com"))
	l.V(0).Infof("\n%s 🔑", logger.Bold("Invest in Data, Harvest over Time."))
}
