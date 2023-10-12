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

package baremetal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

func TestRuntimeManager(t *testing.T) {
	const (
		clusterName = "test"
		homeDir     = "test_data/runtime"
	)
	rm := NewRuntimeManager(clusterName, homeDir)
	rm.configDirsAndPaths()

	err := rm.createDirs()
	assert.NoError(t, err)
	defer func() {
		err := fileutils.DeleteDirIfExists(homeDir)
		assert.NoError(t, err)
	}()

	expect := config.DefaultConfig()
	err = rm.createPaths(expect)
	assert.NoError(t, err)

	cnt, err := os.ReadFile(rm.clusterConfigPath)
	assert.NoError(t, err)

	var actual config.RuntimeConfig
	err = yaml.Unmarshal(cnt, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expect, actual.Config)
}
