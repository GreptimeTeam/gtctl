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

package cluster

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster/create"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster/delete"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster/get"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster/list"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster/scale"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

func NewClusterCommand(l logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Manage GreptimeDB cluster",
		Long:  `Manage GreptimeDB cluster in Kubernetes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}

	cmd.AddCommand(create.NewCreateClusterCommand(l))
	cmd.AddCommand(delete.NewDeleteClusterCommand(l))
	cmd.AddCommand(scale.NewScaleClusterCommand(l))
	cmd.AddCommand(get.NewGetClusterCommand(l))
	cmd.AddCommand(list.NewListClustersCommand(l))

	return cmd
}
