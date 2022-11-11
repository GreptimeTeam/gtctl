package cluster

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/cmd/app/cluster/create"
	"github.com/GreptimeTeam/gtctl/cmd/app/cluster/delete"
	"github.com/GreptimeTeam/gtctl/cmd/app/cluster/get"
	"github.com/GreptimeTeam/gtctl/cmd/app/cluster/list"
	"github.com/GreptimeTeam/gtctl/cmd/app/cluster/scale"
	"github.com/GreptimeTeam/gtctl/pkg/log"
)

func NewClusterCommand() *cobra.Command {
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

	// TODO(zyy17): Maybe ugly to use NewLogger in every command.
	cmd.AddCommand(create.NewCreateClusterCommand(log.NewLogger()))
	cmd.AddCommand(delete.NewDeleteClusterCommand(log.NewLogger()))
	cmd.AddCommand(scale.NewScaleClusterCommand(log.NewLogger()))
	cmd.AddCommand(get.NewGetClusterCommand(log.NewLogger()))
	cmd.AddCommand(list.NewListClustersCommand(log.NewLogger()))

	return cmd
}
