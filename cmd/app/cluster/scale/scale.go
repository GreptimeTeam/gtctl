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

package scale

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	greptimev1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type scaleCliOptions struct {
	Namespace     string
	ComponentType string
	Replicas      int32
	Timeout       int
}

func NewScaleClusterCommand(l log.Logger) *cobra.Command {
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

			if options.ComponentType != string(greptimev1alpha1.FrontendComponentKind) &&
				options.ComponentType != string(greptimev1alpha1.DatanodeComponentKind) {
				return fmt.Errorf("component type is invalid")
			}

			if options.Replicas < 1 {
				return fmt.Errorf("replicas should be equal or greater than 1")
			}

			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
				namespace   = options.Namespace
			)

			cluster, err := m.GetCluster(ctx, &manager.GetClusterOptions{
				ClusterName: clusterName,
				Namespace:   namespace,
			})
			if err != nil && errors.IsNotFound(err) {
				l.Infof("cluster %s in %s not found\n", clusterName, namespace)
				return nil
			}
			if err != nil {
				return err
			}

			var oldReplicas int32
			if options.ComponentType == string(greptimev1alpha1.FrontendComponentKind) {
				oldReplicas = cluster.Spec.Frontend.Replicas
				cluster.Spec.Frontend.Replicas = options.Replicas
			}

			if options.ComponentType == string(greptimev1alpha1.DatanodeComponentKind) {
				oldReplicas = cluster.Spec.Datanode.Replicas
				cluster.Spec.Datanode.Replicas = options.Replicas
			}

			l.Infof("Scaling cluster %s in %s... from %d to %d\n", clusterName, namespace, oldReplicas, options.Replicas)

			if err := m.UpdateCluster(ctx, &manager.UpdateClusterOptions{
				ClusterName: clusterName,
				Namespace:   namespace,
				NewCluster:  cluster,
				Timeout:     time.Duration(options.Timeout) * time.Second,
			}); err != nil {
				return err
			}

			l.Infof("Scaling cluster %s in %s is OK!\n", clusterName, namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ComponentType, "component-type", "c", "", "Component of GreptimeDB cluster, can be 'frontend' and 'datanode'.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}
