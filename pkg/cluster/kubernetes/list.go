// Copyright 2024 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"
	"fmt"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/olekukonko/tablewriter"
	"k8s.io/apimachinery/pkg/api/errors"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
)

func (c *Cluster) List(ctx context.Context, options *opt.ListOptions) error {
	clusters, err := c.list(ctx)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) || clusters == nil {
		return fmt.Errorf("clusters not found")
	}

	c.renderListView(options.Table, clusters)

	return nil
}

func (c *Cluster) list(ctx context.Context) (*greptimedbclusterv1alpha1.GreptimeDBClusterList, error) {
	clusters, err := c.client.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (c *Cluster) configListView(table *tablewriter.Table) {
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

func (c *Cluster) renderListView(table *tablewriter.Table, data *greptimedbclusterv1alpha1.GreptimeDBClusterList) {
	c.configListView(table)

	table.SetHeader([]string{"Name", "Namespace", "Creation Date"})
	defer table.Render()

	for _, cluster := range data.Items {
		table.Append([]string{
			cluster.Name,
			cluster.Namespace,
			cluster.CreationTimestamp.String(),
		})
	}
}
