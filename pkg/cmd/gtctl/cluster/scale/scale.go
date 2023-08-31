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

package scale

import (
	"context"
	"fmt"
	"time"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type scaleCliOptions struct {
	Namespace     string
	ComponentType string
	Replicas      int32
	Timeout       int
}

func NewScaleClusterCommand(l logger.Logger) *cobra.Command {
	var options scaleCliOptions

	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Scale GreptimeDB cluster",
		Long:  `Scale GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			if err := validateScaleOptions(options); err != nil {
				return err
			}

			var (
				ctx = context.Background()
				nn  = types.NamespacedName{
					Namespace: options.Namespace,
					Name:      args[0],
				}
				cancel context.CancelFunc
			)

			if options.Timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Second)
				defer cancel()
			}

			if err := scaleClusterForKubernetes(ctx, options, l, nn); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ComponentType, "component", "c", "", "Component of GreptimeDB cluster, can be 'frontend', 'datanode' and 'meta'.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}

func validateScaleOptions(options scaleCliOptions) error {
	if options.ComponentType == "" {
		return fmt.Errorf("component type is required")
	}

	if options.ComponentType != string(greptimedbclusterv1alpha1.FrontendComponentKind) &&
		options.ComponentType != string(greptimedbclusterv1alpha1.DatanodeComponentKind) &&
		options.ComponentType != string(greptimedbclusterv1alpha1.MetaComponentKind) {
		return fmt.Errorf("component type is invalid")
	}

	if options.Replicas < 1 {
		return fmt.Errorf("replicas should be equal or greater than 1")
	}

	return nil
}

func scaleClusterForKubernetes(ctx context.Context, options scaleCliOptions, l logger.Logger, nn types.NamespacedName) error {
	k8sDeployer, err := k8s.NewDeployer(l)
	if err != nil {
		return err
	}

	cluster, err := k8sDeployer.GetGreptimeDBCluster(ctx, nn.String(), nil)
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

	var oldReplicas int32
	switch options.ComponentType {
	case string(greptimedbclusterv1alpha1.FrontendComponentKind):
		oldReplicas = rawCluster.Spec.Frontend.Replicas
		rawCluster.Spec.Frontend.Replicas = options.Replicas
	case string(greptimedbclusterv1alpha1.DatanodeComponentKind):
		oldReplicas = rawCluster.Spec.Datanode.Replicas
		rawCluster.Spec.Datanode.Replicas = options.Replicas
	case string(greptimedbclusterv1alpha1.MetaComponentKind):
		oldReplicas = rawCluster.Spec.Meta.Replicas
		rawCluster.Spec.Meta.Replicas = options.Replicas
	}

	l.V(0).Infof("Scaling cluster %s in %s... from %d to %d\n", nn.Name, nn.Namespace, oldReplicas, options.Replicas)

	if err = k8sDeployer.UpdateGreptimeDBCluster(ctx, nn.String(), &deployer.UpdateGreptimeDBClusterOptions{
		NewCluster: &deployer.GreptimeDBCluster{Raw: rawCluster},
	}); err != nil {
		return err
	}

	l.V(0).Infof("Scaling cluster %s in %s is OK!\n", nn.Name, nn.Namespace)

	return nil
}
