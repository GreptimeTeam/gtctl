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

package list

import (
	"context"
	"fmt"
	"os"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

func NewListClustersCommand(l logger.Logger) *cobra.Command {
	table := tablewriter.NewWriter(os.Stdout)
	configClustersTableView(table)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all GreptimeDB clusters",
		Long:  `List all GreptimeDB clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if err := listClustersFromKubernetes(ctx, l, table); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func listClustersFromKubernetes(ctx context.Context, l logger.Logger, table *tablewriter.Table) error {
	k8sDeployer, err := k8s.NewDeployer(l)
	if err != nil {
		return err
	}

	clusters, err := k8sDeployer.ListGreptimeDBClusters(ctx, nil)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) || (clusters != nil && len(clusters) == 0) {
		l.Error("clusters not found\n")
		return nil
	}

	if err := renderClustersTableView(table, clusters); err != nil {
		return err
	}

	return nil
}

func configClustersTableView(table *tablewriter.Table) {
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
}

func renderClustersTableView(table *tablewriter.Table, clusters []*deployer.GreptimeDBCluster) error {
	table.SetHeader([]string{"Name", "Namespace", "Creation Date"})

	for _, cluster := range clusters {
		rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
		if !ok {
			return fmt.Errorf("invalid cluster type")
		}
		table.Append([]string{
			rawCluster.Name,
			rawCluster.Namespace,
			rawCluster.CreationTimestamp.String(),
		})
	}

	table.Render()

	return nil
}
