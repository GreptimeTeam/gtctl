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

package list

import (
	"context"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

func NewListClustersCommand(l log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all GreptimeDB clusters",
		Long:  `List all GreptimeDB clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			clusters, err := m.ListClusters(ctx, &manager.ListClusterOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			} else if errors.IsNotFound(err) || (clusters != nil && len(clusters.Items) == 0) {
				l.Infof("clusters not found\n")
				return nil
			}

			// TODO(zyy17): more human friendly output format.
			for _, cluster := range clusters.Items {
				l.Infof("Cluster '%s' in '%s' namespace is running, create at %s\n", cluster.Name, cluster.Namespace, cluster.CreationTimestamp)
			}

			return nil
		},
	}

	return cmd
}
