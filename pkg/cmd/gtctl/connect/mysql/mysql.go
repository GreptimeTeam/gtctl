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

package mysql

import (
	"errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	UpSeparator    = ":"
	AtSeparator    = "@"
	ArgHost        = "-h"
	ArgPort        = "-P"
	ArgUser        = "-u"
	ArgPassword    = "-p"
	MySQL          = "mysql"
	SplitSeparator = "://"
)

func ConnectCommand(cmd *cobra.Command, args []string) error {
	s := args[0]
	suffix := strings.Split(s, SplitSeparator)[1]
	split := strings.Split(suffix, AtSeparator)
	if len(split) != 2 {
		return errors.New("invalid argument, you can try gtctl connect mysql://user:password@host:port")
	}
	up := strings.Split(split[0], UpSeparator)
	if len(up) != 2 {
		return errors.New("invalid argument, you can try gtctl connect mysql://user:password@host:port")
	}
	hp := strings.Split(split[1], UpSeparator)
	if len(hp) != 2 {
		return errors.New("invalid argument, you can try gtctl connect mysql://user:password@host:port")
	}
	user := up[0]
	password := up[1]
	host := hp[0]
	port := hp[1]
	return Connect(user, password, host, port)
}

func Connect(user, password, host, port string) error {
	cmd := exec.Command(MySQL, ArgHost, host, ArgPort, port, ArgUser, user, ArgPassword, password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()
	return err
}
