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
	"time"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/api/scale"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/kubernetes"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type clusterScaleCliOptions struct {
	Namespace     string
	ComponentType string
	Replicas      int32
	Timeout       int
}

func (s clusterScaleCliOptions) validate() error {
	if s.ComponentType == "" {
		return fmt.Errorf("component type is required")
	}

	if s.ComponentType != string(greptimedbclusterv1alpha1.FrontendComponentKind) &&
		s.ComponentType != string(greptimedbclusterv1alpha1.DatanodeComponentKind) &&
		s.ComponentType != string(greptimedbclusterv1alpha1.MetaComponentKind) {
		return fmt.Errorf("component type is invalid")
	}

	if s.Replicas < 0 {
		return fmt.Errorf("replicas should be equal or greater than 0")
	}

	return nil
}

func NewScaleClusterCommand(l logger.Logger) *cobra.Command {
	var options clusterScaleCliOptions

	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Scale GreptimeDB cluster",
		Long:  `Scale GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			if err := options.validate(); err != nil {
				return err
			}

			var (
				ctx    = context.Background()
				err    error
				scaler scale.Scaler
				cancel context.CancelFunc
			)

			if options.Timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Second)
				defer cancel()
			}

			scaler, err = kubernetes.NewCluster(l)
			if err != nil {
				return err
			}

			scaleOptions := &scale.Options{
				Name:          args[0],
				Namespace:     options.Namespace,
				NewReplicas:   options.Replicas,
				ComponentType: options.ComponentType,
			}
			return scaler.Scale(ctx, scaleOptions)
		},
	}

	cmd.Flags().StringVarP(&options.ComponentType, "component", "c", "", "Component of GreptimeDB cluster, can be 'frontend', 'datanode' and 'meta'.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", 300, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}
