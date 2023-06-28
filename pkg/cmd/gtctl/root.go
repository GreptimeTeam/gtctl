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

package gtctl

import (
	"fmt"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/connect"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/constants"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/version"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	internalversion "github.com/GreptimeTeam/gtctl/pkg/version"
)

type flagArgs struct {
	Verbosity int32
}

func NewRootCommand() *cobra.Command {
	var (
		flags = &flagArgs{}

		l = logger.New(os.Stdout, log.Level(flags.Verbosity), logger.WithColored())
	)

	cmd := &cobra.Command{
		Args:         cobra.NoArgs,
		Use:          "gtctl",
		Short:        "gtctl is a command-line tool for managing GreptimeDB cluster.",
		Long:         fmt.Sprintf("%s\ngtctl is a command-line tool for managing GreptimeDB cluster.", constants.GtctlTextBanner),
		Version:      internalversion.Get().String(),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(l, flags, cmd)
		},
	}

	cmd.PersistentFlags().Int32VarP(
		&flags.Verbosity,
		"verbosity",
		"v",
		0,
		"info log verbosity, higher value produces more output",
	)

	// Add all top level subcommands.
	cmd.AddCommand(version.NewVersionCommand(l))
	cmd.AddCommand(cluster.NewClusterCommand(l))
	cmd.AddCommand(connect.NewConnectCommand(l))

	return cmd
}

func runE(logger log.Logger, flags *flagArgs, _ *cobra.Command) error {
	return maybeSetVerbosity(logger, log.Level(flags.Verbosity))
}

// maybeSetVerbosity will call logger.SetVerbosity(verbosity) if logger has a SetVerbosity method.
func maybeSetVerbosity(logger log.Logger, verbosity log.Level) error {
	type verboser interface {
		SetVerbosity(log.Level)
	}
	v, ok := logger.(verboser)
	if ok {
		v.SetVerbosity(verbosity)
		return nil
	}

	return fmt.Errorf("logger does not implement SetVerbosity")
}
