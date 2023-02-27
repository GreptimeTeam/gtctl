// Copyright 2022 Greptime Team
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

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/cmd/app/cluster"
	"github.com/GreptimeTeam/gtctl/cmd/app/version"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	internalversion "github.com/GreptimeTeam/gtctl/pkg/version"
)

const gtctlTextBanner = "          __       __  __\n   ____ _/ /______/ /_/ /\n  / __ `/ __/ ___/ __/ / \n / /_/ / /_/ /__/ /_/ /  \n \\__, /\\__/\\___/\\__/_/   \n/____/   \n"

type flagArgs struct {
	Verbosity int32
}

func NewRootCommand() *cobra.Command {
	flags := &flagArgs{}
	cmd := &cobra.Command{
		Args:    cobra.NoArgs,
		Use:     "gtctl",
		Short:   "gtctl is a command-line tool for managing GreptimeDB cluster.",
		Long:    fmt.Sprintf("%s\ngtctl is a command-line tool for managing GreptimeDB cluster.", gtctlTextBanner),
		Version: internalversion.Get().String(),
	}

	cmd.PersistentFlags().Int32VarP(
		&flags.Verbosity,
		"verbosity",
		"v",
		0,
		"info log verbosity, higher value produces more output",
	)

	l := logger.New(os.Stdout, log.Level(flags.Verbosity), logger.WithColored())

	// Add all top level subcommands.
	cmd.AddCommand(version.NewVersionCommand(l))
	cmd.AddCommand(cluster.NewClusterCommand(l))

	return cmd
}
