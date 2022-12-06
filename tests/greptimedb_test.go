package tests

import (
	"context"
	"database/sql"
	"fmt"
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
                        TIME INDEX (ts)
                     )
                     PARTITION BY RANGE COLUMNS (n) (
    				 	PARTITION r0 VALUES LESS THAN (5),
    					PARTITION r1 VALUES LESS THAN (9),
    					PARTITION r2 VALUES LESS THAN (MAXVALUE),
					)
					engine=mito`

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

var _ = Describe("Testing of greptimedb", func() {
	It("Sql test of basic cluster", func() {
		go func() {
			forwardRequest()
		}()

		By("Connecting GreptimeDB")
		var db *sql.DB
		var conn *sql.Conn
		var err error

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
	})
})

func forwardRequest() {
	for {
		cmd := exec.Command("kubectl", "port-forward", "svc/mydb-frontend", "4002:4002")
		if err := cmd.Run(); err != nil {
			klog.Errorf("Failed to port forward:%v", err)
			return
		}
	}
}
