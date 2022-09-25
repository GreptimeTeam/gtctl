package create

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "create",
		Short: "Create GreptimeDB cluster.",
		Long:  `Create GreptimeDB cluster in Kubernetes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}

	cmd.AddCommand(NewCreateClusterCommand())

	return cmd
}
