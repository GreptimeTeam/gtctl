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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
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

			if options.ComponentType == "" {
				return fmt.Errorf("component type is required")
			}

			if options.ComponentType != string(greptimedbclusterv1alpha1.FrontendComponentKind) &&
				options.ComponentType != string(greptimedbclusterv1alpha1.DatanodeComponentKind) {
				return fmt.Errorf("component type is invalid")
			}

			if options.Replicas < 1 {
				return fmt.Errorf("replicas should be equal or greater than 1")
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
			if err != nil {
				return err
			}

			rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
			if !ok {
				return fmt.Errorf("invalid cluster type")
			}

			var oldReplicas int32
			if options.ComponentType == string(greptimedbclusterv1alpha1.FrontendComponentKind) {
				oldReplicas = rawCluster.Spec.Frontend.Replicas
				rawCluster.Spec.Frontend.Replicas = options.Replicas
			}

			if options.ComponentType == string(greptimedbclusterv1alpha1.DatanodeComponentKind) {
				oldReplicas = rawCluster.Spec.Datanode.Replicas
				rawCluster.Spec.Datanode.Replicas = options.Replicas
			}

			l.V(0).Infof("Scaling cluster %s in %s... from %d to %d\n", clusterName, namespace, oldReplicas, options.Replicas)

			if err := k8sDeployer.UpdateGreptimeDBCluster(ctx, name, &deployer.UpdateGreptimeDBClusterOptions{
				NewCluster: &deployer.GreptimeDBCluster{Raw: rawCluster},
			}); err != nil {
				return err
			}

			l.V(0).Infof("Scaling cluster %s in %s is OK!\n", clusterName, namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ComponentType, "component-type", "c", "", "Component of GreptimeDB cluster, can be 'frontend' and 'datanode'.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}
