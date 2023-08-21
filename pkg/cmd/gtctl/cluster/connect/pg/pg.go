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

package pg

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/go-pg/pg/v10"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	PostgresSQL = "psql"
	DbName      = "public"
	DefaultAddr = "127.0.0.1"
	DefaultNet  = "tcp"
	ArgsHost    = "-h"
	ArgsPort    = "-p"
	ArgDb       = "-d"
	PrePort     = ":"
	Kubectl     = "kubectl"
	PortForward = "port-forward"
)

func ConnectCommand(rawCluster *greptimedbclusterv1alpha1.GreptimeDBCluster, l logger.Logger) error {
	return connect(strconv.Itoa(int(rawCluster.Spec.PostgresServicePort)), rawCluster.Name, l)
}

func connect(port, clusterName string, l logger.Logger) error {
	waitGroup := sync.WaitGroup{}
	cmd := exec.CommandContext(context.Background(), Kubectl, PortForward, "-n", "default", "svc/"+clusterName+"-frontend", port+PrePort+port)
	err := cmd.Start()
	if err != nil {
		l.Errorf("Error starting port-forwarding: %v", err)
		return err
	}
	defer func() {
		if recover() != nil {
			err := cmd.Process.Kill()
			if err != nil {
				l.Errorf("Error killing port-forwarding process: %v", err)
			}
		}
	}()
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		if err = cmd.Wait(); err != nil {
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
		opt := &pg.Options{
			Addr:     DefaultAddr + PrePort + port,
			Network:  DefaultNet,
			Database: DbName,
		}
		db := pg.Connect(opt)
		_, err = db.Exec("SELECT 1")
		if err == nil {
			break
		}
	}
	cmd = pgCommand(port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Start()
	if err != nil {
		l.Errorf("Error starting pg: %v", err)
		return err
	}
	if err = cmd.Wait(); err != nil {
		l.Errorf("Error waiting for pg client to finish: %v", err)
		return err
	}
	// gracefully stop port-forwarding
	err = cmd.Process.Kill()
	if err != nil {
		if err == os.ErrProcessDone {
			l.V(1).Info("Shutting down port-forwarding successfully")
			return nil
		}
		return err
	}
	waitGroup.Wait()
	return nil
}

func pgCommand(port string) *exec.Cmd {
	return exec.Command(PostgresSQL, ArgsHost, DefaultAddr, ArgsPort, port, ArgDb, DbName)
}
