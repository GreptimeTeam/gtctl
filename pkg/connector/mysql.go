/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package connector

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/go-sql-driver/mysql"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	mySQLDriver      = "mysql"
	mySQLDefaultAddr = "127.0.0.1"
	mySQLDefaultNet  = "tcp"

	mySQLPortArg = "-P"
	mySQLHostArg = "-h"

	kubectl     = "kubectl"
	portForward = "port-forward"
)

// Mysql connects to a GreptimeDB cluster using mysql protocol.
func Mysql(port, clusterName string, l logger.Logger) error {
	waitGroup := sync.WaitGroup{}

	// TODO: is there any elegant way to enable port-forward?
	cmd := exec.CommandContext(context.Background(), kubectl, portForward, "-n", "default", "svc/"+clusterName+"-frontend", fmt.Sprintf("%s:%s", port, port))
	if err := cmd.Start(); err != nil {
		l.Errorf("Error starting port-forwarding: %v", err)
		return err
	}

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		if err := cmd.Wait(); err != nil {
			// exit status 1
			exitError, ok := err.(*exec.ExitError)
			if !ok {
				l.Errorf("Error waiting for port-forwarding to finish: %v", err)
				return
			}
			if exitError.Sys().(syscall.WaitStatus).ExitStatus() == 1 {
				return
			}
		}
	}()

	for {
		cfg := mysql.Config{
			Net:                  mySQLDefaultNet,
			Addr:                 net.JoinHostPort(mySQLDefaultAddr, port),
			User:                 "",
			Passwd:               "",
			DBName:               "",
			AllowNativePasswords: true,
		}

		db, err := sql.Open(mySQLDriver, cfg.FormatDSN())
		if err != nil {
			continue
		}

		if _, err = db.Conn(context.Background()); err != nil {
			continue
		}

		if err = db.Close(); err != nil {
			if err == os.ErrProcessDone {
				return nil
			}
			return err
		}

		break
	}

	cmd = mysqlCommand(port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		l.Errorf("Error starting mysql client: %v", err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		l.Errorf("Error waiting for mysql client to finish: %v", err)
		return err
	}

	// gracefully stop port-forwarding
	if err := cmd.Process.Kill(); err != nil {
		if err == os.ErrProcessDone {
			l.V(1).Info("Shutting down port-forwarding successfully")
			return nil
		}
		return err
	}

	waitGroup.Wait()
	return nil
}

func mysqlCommand(port string) *exec.Cmd {
	return exec.Command(mySQLDriver, mySQLHostArg, mySQLDefaultAddr, mySQLPortArg, port)
}
