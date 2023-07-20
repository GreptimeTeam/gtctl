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
	"gopkg.in/yaml.v3"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name   string
	expect bool
	errmsg string
}

var testCases = []testCase{
	{
		name:   "default-config",
		expect: true,
	},
	{
		name:   "config-with-nil-part",
		expect: false,
		errmsg: "error at field `.Cluster.Meta`: got nil",
	},
	{
		name:   "config-with-empty-part",
		expect: false,
		errmsg: "error at field `.Cluster.Datanode.HTTPAddr`: got empty",
	},
	{
		name:   "config-with-invalid-addr",
		expect: false,
		errmsg: "error at field `.Cluster.Datanode.HTTPAddr`: invalid ip address '12345.0.0.0'",
	},
}

func TestValidateConfig(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(fmt.Sprintf("testdata/%s.yaml", tc.name))
			assert.NoError(t, err)

			var config *Config
			err = yaml.Unmarshal(content, &config)
			assert.NoError(t, err)

			err = ValidateConfig(config)
			if tc.expect {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.errmsg, "got wrong error")
			}
		})
	}
}
