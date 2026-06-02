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

package file

import (
	"bytes"
	"errors"
	"os"
	"path"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type errorCloser struct{}

func (errorCloser) Close() error {
	return errors.New("close failed")
}

func TestUncompress(t *testing.T) {
	const (
		testContent = "helloworld"
		outputDir   = "testdata/output"
	)

	tests := []struct {
		filename string
		path     string
		dst      string
	}{
		{"test-zip", "testdata/test-zip.zip", outputDir},
		{"test-tgz", "testdata/test-tgz.tgz", outputDir},
		{"test-tgz", "testdata/test-tar-gz.tar.gz", outputDir},
	}

	// Clean up output dir.
	defer RemoveAll(outputDir)

	for _, test := range tests {
		if err := Uncompress(test.path, test.dst); err != nil {
			t.Errorf("uncompress file '%s': %v", test.path, err)
		}

		dataFile := path.Join(test.dst, test.filename, "data")
		data, err := os.ReadFile(dataFile)
		if err != nil {
			t.Errorf("read file '%s': %v", dataFile, err)
		}

		if string(data) != testContent {
			t.Errorf("file content is not '%s': %s", testContent, string(data))
		}
	}
}

func TestMustCloseLogsError(t *testing.T) {
	var buf bytes.Buffer
	previous := logger.Default()
	logger.SetDefault(logger.New(&buf, log.Level(0)))
	defer logger.SetDefault(previous)

	MustClose(errorCloser{})

	if got := buf.String(); !strings.Contains(got, "failed to close: close failed") {
		t.Fatalf("expected close error to be logged, got %q", got)
	}
}
