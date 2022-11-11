package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/cmd/app/cluster"
	"github.com/GreptimeTeam/gtctl/cmd/app/version"
	internalversion "github.com/GreptimeTeam/gtctl/pkg/version"
)

const gtctlTextBanner = "          __       __  __\n   ____ _/ /______/ /_/ /\n  / __ `/ __/ ___/ __/ / \n / /_/ / /_/ /__/ /_/ /  \n \\__, /\\__/\\___/\\__/_/   \n/____/   \n"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:    cobra.NoArgs,
		Use:     "gtctl",
		Short:   "gtctl is a command-line tool for managing GreptimeDB cluster.",
		Long:    fmt.Sprintf("%s\ngtctl is a command-line tool for managing GreptimeDB cluster.", gtctlTextBanner),
		Version: internalversion.Get().String(),
	}

	// Add all top level subcommands.
	cmd.AddCommand(version.NewVersionCommand())
	cmd.AddCommand(cluster.NewClusterCommand())

	return cmd
}
