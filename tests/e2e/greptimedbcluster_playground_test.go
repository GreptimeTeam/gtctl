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
)

var _ = Describe("Basic test of greptimedb cluster playground", func() {
	It("Bootstrap cluster", func() {
		var err error
		go func() {
			err = playground()
			Expect(err).NotTo(HaveOccurred(), "failed to create cluster playground")
		}()

		By("Connecting GreptimeDB")
		var db *sql.DB
		var conn *sql.Conn

		Eventually(func() error {
			cfg := mysql.Config{
				Net:                  "tcp",
				Addr:                 "127.0.0.1:4200",
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
	})
})

func playground() error {
	cmd := exec.Command("../../bin/gtctl", "playground", "--config", "testdata/cluster.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
