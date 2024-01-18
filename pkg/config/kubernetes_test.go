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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config []string
		expect SetValues
		err    bool
	}{
		{
			name:   "all-with-prefix",
			config: []string{"cluster.foo=bar", "etcd.foo=bar", "operator.foo=bar"},
			expect: SetValues{
				ClusterConfig:  "foo=bar",
				EtcdConfig:     "foo=bar",
				OperatorConfig: "foo=bar",
			},
		},
		{
			name:   "all-without-prefix",
			config: []string{"foo=bar", "foo.boo=bar", "foo.boo.coo=bar"},
			expect: SetValues{
				ClusterConfig: "foo=bar,foo.boo=bar,foo.boo.coo=bar",
			},
		},
		{
			name:   "mix-with-prefix",
			config: []string{"etcd.foo=bar", "foo.boo=bar", "foo.boo.coo=bar"},
			expect: SetValues{
				ClusterConfig: "foo.boo=bar,foo.boo.coo=bar",
				EtcdConfig:    "foo=bar",
			},
		},
		{
			name:   "empty-values",
			config: []string{""},
			err:    true,
		},
		{
			name:   "empty-config",
			config: []string{},
			expect: SetValues{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SetValues{RawConfig: tc.config}
			err := actual.Parse()

			if tc.err {
				assert.Error(t, err)
				return
			}

			tc.expect.RawConfig = tc.config
			assert.Equal(t, tc.expect, actual)
		})
	}
}
