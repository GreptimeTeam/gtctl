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

package connect

import (
	"errors"
	"github.com/GreptimeTeam/gtctl/pkg/cmd/gtctl/connect/mysql"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/spf13/cobra"
	"strings"
)

const (
	SplitSeparator = "://"
	MySQL          = "mysql"
)

func NewConnectCommand(l logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a GreptimeDB cluster",
		Long:  `Connect to a GreptimeDB cluster`,
		RunE:  connectCommand,
	}
	return cmd
}

func connectCommand(cmd *cobra.Command, args []string) error {
	s := args[0]
	split := strings.Split(s, SplitSeparator)
	if len(split) != 2 {
		return errors.New("invalid argument, you can try gtctl connect mysql://user:password@host:port")
	}
	prefix := split[0]
	switch prefix {
	case MySQL:
		return mysql.ConnectCommand(cmd, args)
	default:
		return errors.New("invalid argument, you can try gtctl connect mysql://user:password@host:port")
	}
}
