// Copyright 2024 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-sql-driver/mysql"
	"k8s.io/klog/v2"
)

const (
	createTableSQL = `CREATE TABLE dist_table (
                        ts TIMESTAMP DEFAULT current_timestamp(),
                        n INT,
    					row_id INT,
                        TIME INDEX (ts),
                        PRIMARY KEY(n)
                     )
                     PARTITION BY RANGE COLUMNS (n) (
    				 	PARTITION r0 VALUES LESS THAN (5),
    					PARTITION r1 VALUES LESS THAN (9),
    					PARTITION r2 VALUES LESS THAN (MAXVALUE),
					)`

	insertDataSQLStr = "INSERT INTO dist_table(n, row_id) VALUES (%d, %d)"

	selectDataSQL = `SELECT * FROM dist_table`

	testRowIDNum = 10
)

var (
	defaultQueryTimeout = 5 * time.Second
)

// TestData is the schema of test data in SQL table.
type TestData struct {
	timestamp string
	n         int32
	rowID     int32
}

var _ = Describe("Basic test of greptimedb cluster", func() {
	It("Bootstrap cluster", func() {
		var err error
		err = createCluster()
		Expect(err).NotTo(HaveOccurred(), "failed to create cluster")

		err = getCluster()
		Expect(err).NotTo(HaveOccurred(), "failed to get cluster")

		err = listCluster()
		Expect(err).NotTo(HaveOccurred(), "failed to list cluster")

		go func() {
			forwardRequest()
		}()

		By("Connecting GreptimeDB")
		var db *sql.DB
		var conn *sql.Conn

		Eventually(func() error {
			cfg := mysql.Config{
				Net:                  "tcp",
				Addr:                 "127.0.0.1:4002",
				User:                 "",
				Passwd:               "",
				DBName:               "",
				AllowNativePasswords: true,
			}

			db, err = sql.Open("mysql", cfg.FormatDSN())
			if err != nil {
				return err
			}

			conn, err = db.Conn(context.TODO())
			if err != nil {
				return err
			}

			return nil
		}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

		By("Execute SQL queries after connecting")

		ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()

		_, err = conn.ExecContext(ctx, createTableSQL)
		Expect(err).NotTo(HaveOccurred(), "failed to create SQL table")

		ctx, cancel = context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()
		for rowID := 1; rowID <= testRowIDNum; rowID++ {
			insertDataSQL := fmt.Sprintf(insertDataSQLStr, rowID, rowID)
			_, err = conn.ExecContext(ctx, insertDataSQL)
			Expect(err).NotTo(HaveOccurred(), "failed to insert data")
		}

		ctx, cancel = context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()
		results, err := conn.QueryContext(ctx, selectDataSQL)
		Expect(err).NotTo(HaveOccurred(), "failed to get data")

		var data []TestData
		for results.Next() {
			var d TestData
			err = results.Scan(&d.timestamp, &d.n, &d.rowID)
			Expect(err).NotTo(HaveOccurred(), "failed to scan data that query from db")
			data = append(data, d)
		}
		Expect(len(data) == testRowIDNum).Should(BeTrue(), "get the wrong data from db")

		err = deleteCluster()
		Expect(err).NotTo(HaveOccurred(), "failed to delete cluster")
	})
})

func createCluster() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "create", "mydb", "--timeout", "300")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getCluster() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "get", "mydb")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func listCluster() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "list")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func deleteCluster() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "delete", "mydb", "--tear-down-etcd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func forwardRequest() {
	for {
		cmd := exec.Command("kubectl", "port-forward", "svc/mydb-frontend", "4002:4002")
		if err := cmd.Run(); err != nil {
			klog.Errorf("Failed to port forward: %v", err)
			return
		}
	}
}
