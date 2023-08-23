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

package get

import (
	"context"
	"fmt"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal"
	bmconfig "github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type getClusterCliOptions struct {
	Namespace string

	// The options for getting GreptimeDBCluster in bare-metal.
	BareMetal bool
}

func NewGetClusterCommand(l logger.Logger) *cobra.Command {
	var options getClusterCliOptions
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get GreptimeDB cluster",
		Long:  `Get GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
			)

			nn := types.NamespacedName{
				Namespace: options.Namespace,
				Name:      clusterName,
			}

			if options.BareMetal {
				return getClusterFromBareMetal(ctx, l, nn)
			} else {
				return getClusterFromKubernetes(ctx, l, nn)
			}
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.BareMetal, "bare-metal", false, "Get the greptimedb cluster on bare-metal environment.")

	return cmd
}

func getClusterFromKubernetes(ctx context.Context, l logger.Logger, nn types.NamespacedName) error {
	deployer, err := k8s.NewDeployer(l)
	if err != nil {
		return err
	}

	cluster, err := deployer.GetGreptimeDBCluster(ctx, nn.String(), nil)
	if err != nil && errors.IsNotFound(err) {
		l.Errorf("cluster %s in %s not found\n", nn.Name, nn.Namespace)
		return nil
	}
	if err != nil {
		return err
	}

	rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
	if !ok {
		return fmt.Errorf("invalid cluster type")
	}

	l.V(0).Infof("Cluster '%s' in '%s' namespace is running, create at %s\n",
		rawCluster.Name, rawCluster.Namespace, rawCluster.CreationTimestamp)
	return nil
}

func getClusterFromBareMetal(ctx context.Context, l logger.Logger, nn types.NamespacedName) error {
	deployer, err := baremetal.NewDeployer(l, nn.Name, baremetal.WithCreateNoDirs())
	if err != nil {
		return nil
	}

	cluster, err := deployer.GetGreptimeDBCluster(ctx, nn.Name, nil)
	if err != nil {
		return err
	}

	rawCluster, ok := cluster.Raw.(*bmconfig.Config)
	if !ok {
		return fmt.Errorf("invalid cluster type")
	}

	l.V(0).Infof("Cluster '%s' in bare-metal is running, create at %s\n",
		nn.Name, rawCluster.Cluster.Artifact.Version)
	return nil
}
