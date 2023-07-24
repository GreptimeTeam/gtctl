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
	"context"
	"database/sql"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/go-sql-driver/mysql"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	ArgHost = "-h"
	ArgPort = "-P"
	MySQL   = "mysql"
)

// ConnectCommand connects to a GreptimeDB cluster
func ConnectCommand(rawCluster *greptimedbclusterv1alpha1.GreptimeDBCluster, l logger.Logger) error {
	return connect("127.0.0.1", strconv.Itoa(int(rawCluster.Spec.MySQLServicePort)), rawCluster.Name, l)
}

// connect connects to a GreptimeDB cluster
func connect(host, port, clusterName string, l logger.Logger) error {
	waitGroup := sync.WaitGroup{}
	cmd := exec.CommandContext(context.Background(), "kubectl", "port-forward", "-n", "default", "svc/"+clusterName+"-frontend", port+":"+port)
	err := cmd.Start()
	if err != nil {
		l.Errorf("Error starting port-forwarding: %v", err)
		return err
	}
	go func() {
		waitGroup.Add(1)
		defer waitGroup.Done()
		if err = cmd.Wait(); err != nil {
		}
	}()
	for {
		cfg := mysql.Config{
			Net:                  "tcp",
			Addr:                 "127.0.0.1:4002",
			User:                 "",
			Passwd:               "",
			DBName:               "",
			AllowNativePasswords: true,
		}

		db, err := sql.Open("mysql", cfg.FormatDSN())
		if err != nil {
			continue
		}

		_, err = db.Conn(context.Background())
		if err != nil {
			continue
		}

		err = db.Close()
		if err != nil {
			l.V(1).Infof("Error closing connection: %v", err)
			return err
		}
		break
	}

	cmd = exec.Command(MySQL, ArgHost, host, ArgPort, port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Start()
	if err != nil {
		l.Errorf("Error starting mysql client: %v", err)
		return err
	}
	if err = cmd.Wait(); err != nil {
		l.Errorf("Error waiting for mysql client to finish: %v", err)
		return err
	}
	// gracefully stop port-forwarding
	err = cmd.Process.Kill()
	if err != nil {
		if err.Error() != "os: process already finished" {
			l.V(1).Info("Shutting down port-forwarding successfully")
		}
		return err
	}
	waitGroup.Wait()
	return nil
}
