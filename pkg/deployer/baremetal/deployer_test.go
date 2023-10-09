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
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

var L = logger.New(os.Stdout, 1, logger.WithColored())

func TestNewDeployer(t *testing.T) {
	homedir := "gtctl-test-new-deployer"
	clusterName := "test"
	defer func() {
		err := fileutils.DeleteDirIfExists(homedir)
		assert.NoError(t, err)
	}()

	// New Deployer with no options
	deployer, err := NewDeployer(L, clusterName, WithBaseDir(homedir))
	assert.NoError(t, err)
	d1, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d1)
	assert.Equal(t, d1.baseDir, homedir)
	assert.Equal(t, d1.clusterDir, path.Join(homedir, clusterName))
	assert.Equal(t, d1.workingDirs, component.WorkingDirs{
		DataDir: path.Join(homedir, clusterName, "data"),
		LogsDir: path.Join(homedir, clusterName, "logs"),
		PidsDir: path.Join(homedir, clusterName, "pids"),
	})
	assert.False(t, d1.enableCache)

	// New Deployer with always download option
	deployer, err = NewDeployer(L, clusterName, WithEnableCache(true))
	assert.NoError(t, err)
	d2, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d2)
	assert.True(t, d2.enableCache)

	// New Deployer with replace config option
	newConfig := config.DefaultConfig()
	newConfig.Cluster.Frontend.Replicas = 3
	deployer, err = NewDeployer(L, clusterName, WithReplaceConfig(newConfig))
	assert.NoError(t, err)
	d3, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d3)
	assert.Equal(t, newConfig, d3.config)

	// New Deployer with specific version
	version := "foobar"
	deployer, err = NewDeployer(L, clusterName, WithGreptimeVersion(version))
	assert.NoError(t, err)
	d4, ok := deployer.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, d4)
	assert.Equal(t, version, d4.config.Cluster.Artifact.Version)
}

func TestDeleteGreptimeClusterForeground(t *testing.T) {
	homedir := "gtctl-test-foreground-delete"
	clusterName := "test"
	defer func() {
		err := fileutils.DeleteDirIfExists(homedir)
		assert.NoError(t, err)
	}()

	d, err := NewDeployer(L, clusterName, WithBaseDir(homedir))
	assert.NoError(t, err)
	deployer, ok := d.(*Deployer)
	assert.True(t, ok)
	assert.NotNil(t, deployer)

	err = deployer.deleteGreptimeDBClusterForeground(context.Background(), component.DeleteOptions{})
	assert.NoError(t, err)

	info, err := os.Stat(deployer.clusterDir)
	if os.IsExist(err) {
		t.Errorf("cluster dir %s should not exist: %v", deployer.clusterDir, err)
	}
	if info != nil && !info.IsDir() {
		t.Errorf("cluster dir %s is not a dir", deployer.clusterDir)
	}
}
