package get

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/log"
)

func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "get",
		Short: "Get GreptimeDB cluster.",
		Long:  `Get GreptimeDB cluster in Kubernetes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}

	cmd.AddCommand(NewGetClusterCommand(log.NewLogger()))
	cmd.AddCommand(NewGetAllClustersCommand(log.NewLogger()))

	return cmd
}
