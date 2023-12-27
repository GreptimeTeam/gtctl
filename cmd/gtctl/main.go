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

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/plugins"
	"github.com/GreptimeTeam/gtctl/pkg/version"
)

func NewRootCommand() *cobra.Command {
	const GtctlTextBanner = `
          __       __  __
   ____ _/ /______/ /_/ /
  / __ '/ __/ ___/ __/ / 
 / /_/ / /_/ /__/ /_/ /  
 \__, /\__/\___/\__/_/   
/____/`

	var (
		verbosity int32

		l = logger.New(os.Stdout, log.Level(verbosity), logger.WithColored())
	)

	cmd := &cobra.Command{
		Use:          "gtctl",
		Short:        "gtctl is a command-line tool for managing GreptimeDB cluster.",
		Long:         fmt.Sprintf("%s\ngtctl is a command-line tool for managing GreptimeDB cluster.", GtctlTextBanner),
		Version:      version.Get().String(),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			type verboser interface {
				SetVerbosity(log.Level)
			}
			if v, ok := l.(verboser); ok {
				v.SetVerbosity(log.Level(verbosity))
				return nil
			}

			return fmt.Errorf("logger does not implement SetVerbosity")
		},
	}

	cmd.PersistentFlags().Int32VarP(&verbosity, "verbosity", "v", 0, "info log verbosity, higher value produces more output")

	// Add all top level subcommands.
	cmd.AddCommand(NewVersionCommand(l))
	cmd.AddCommand(NewClusterCommand(l))
	cmd.AddCommand(NewPlaygroundCommand(l))

	return cmd
}

func main() {
	pm, err := plugins.NewManager()
	if err != nil {
		panic(err)
	}

	if pm.ShouldRun(os.Args[1]) {
		if err = pm.Run(os.Args[1:]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err = NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
