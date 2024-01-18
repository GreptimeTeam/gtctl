/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/cluster/kubernetes"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

func NewListClustersCommand(l logger.Logger) *cobra.Command {
	table := tablewriter.NewWriter(os.Stdout)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all GreptimeDB clusters",
		Long:  `List all GreptimeDB clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var ctx = context.Background()

			cluster, err := kubernetes.NewCluster(l)
			if err != nil {
				return err
			}

			return cluster.List(ctx, &opt.ListOptions{
				GetOptions: opt.GetOptions{
					Table: table,
				},
			})
		},
	}

	return cmd
}
