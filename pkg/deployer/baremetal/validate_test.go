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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
)

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name   string
		expect bool
		errKey []string
	}{
		{
			name:   "valid_config",
			expect: true,
		},
		{
			name:   "invalid_hostname_port",
			expect: false,
			errKey: []string{
				"Config.Cluster.MetaSrv.ServerAddr",
				"Config.Cluster.Datanode.HTTPAddr",
			},
		},
		{
			name:   "invalid_replicas",
			expect: false,
			errKey: []string{
				"Config.Cluster.Frontend.Replicas",
				"Config.Cluster.Datanode.Replicas",
			},
		},
		{
			name:   "invalid_artifact",
			expect: false,
			errKey: []string{
				"Config.Etcd.Artifact.Artifact",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var actual config.Config
			if err := loadConfig(fmt.Sprintf("test_data/%s.yaml", tc.name), &actual); err != nil {
				t.Errorf("error while loading %s file: %v", tc.name, err)
			}

			err := ValidateConfig(&actual)
			if tc.expect {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, key := range tc.errKey {
					assert.Contains(t, err.Error(), key)
				}
			}
		})
	}
}

func loadConfig(path string, ret *config.Config) error {
	configs, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(configs, ret); err != nil {
		return err
	}
	return nil
}
