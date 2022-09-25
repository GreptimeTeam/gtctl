package main

import (
	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/cmd/app/create"
	"github.com/GreptimeTeam/gtctl/cmd/app/delete"
	"github.com/GreptimeTeam/gtctl/cmd/app/scale"
	"github.com/GreptimeTeam/gtctl/cmd/app/version"
	internalversion "github.com/GreptimeTeam/gtctl/pkg/version"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:    cobra.NoArgs,
		Use:     "gtctl",
		Short:   "gtctl is a command-line tool for managing GreptimeDB cluster.",
		Long:    `gtctl is a command-line tool for managing GreptimeDB cluster.`,
		Version: internalversion.Get().String(),
	}

	// Add all top level subcommands.
	cmd.AddCommand(create.NewCreateCommand())
	cmd.AddCommand(version.NewVersionCommand())
	cmd.AddCommand(delete.NewDeleteCommand())
	cmd.AddCommand(scale.NewScaleCommand())

	return cmd
}
