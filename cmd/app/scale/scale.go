package scale

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewScaleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "scale",
		Short: "Scale GreptimeDB cluster.",
		Long:  `Scale GreptimeDB cluster in Kubernetes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}

	cmd.AddCommand(NewScaleClusterCommand())

	return cmd
}
