package delete

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/log"
)

func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "delete",
		Short: "Delete GreptimeDB cluster.",
		Long:  `Delete GreptimeDB cluster in Kubernetes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}

	cmd.AddCommand(NewDeleteClusterCommand(log.NewLogger()))

	return cmd
}
