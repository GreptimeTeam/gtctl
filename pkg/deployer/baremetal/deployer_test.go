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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

var L = logger.New(os.Stdout, 1, logger.WithColored())

func TestNewDeployer(t *testing.T) {
	homedir, err := os.UserHomeDir()
	assert.NoError(t, err)
	clusterName := "test"

	// New Deployer with no options
	deployer, err := NewDeployer(L, clusterName)
	assert.NoError(t, err)
	d1, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d1)
	assert.Equal(t, d1.baseDir, path.Join(homedir, config.GtctlDir))
	assert.Equal(t, d1.clusterDir, path.Join(homedir, config.GtctlDir, clusterName))
	assert.Equal(t, d1.workingDirs, component.WorkingDirs{
		DataDir: path.Join(homedir, config.GtctlDir, clusterName, "data"),
		LogsDir: path.Join(homedir, config.GtctlDir, clusterName, "logs"),
		PidsDir: path.Join(homedir, config.GtctlDir, clusterName, "pids"),
	})
	assert.False(t, d1.alwaysDownload)

	// New Deployer with always download option
	deployer, err = NewDeployer(L, clusterName, WithAlawaysDownload(true))
	assert.NoError(t, err)
	d2, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d2)
	assert.True(t, d2.alwaysDownload)

	// New Deployer with config option
	newConfig := config.DefaultConfig()
	newConfig.Cluster.Frontend.Replicas = 3
	deployer, err = NewDeployer(L, clusterName, WithConfig(newConfig))
	assert.NoError(t, err)
	d3, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d3)
	assert.Equal(t, newConfig, d3.config)
}
