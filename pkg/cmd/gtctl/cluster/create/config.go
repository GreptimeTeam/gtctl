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

package create

import (
	"fmt"
	"strings"
)

const (
	// Various of support config type
	configOperator = "operator"
	configCluster  = "cluster"
	configEtcd     = "etcd"
)

type configValues struct {
	rawConfig []string

	operatorConfig string
	clusterConfig  string
	etcdConfig     string
}

// parseConfig parse raw config values and classify it to different
// categories of config type by its prefix.
func (c *configValues) parseConfig() error {
	var (
		operatorConfig []string
		clusterConfig  []string
		etcdConfig     []string
	)

	for _, raw := range c.rawConfig {
		if len(raw) == 0 {
			return fmt.Errorf("cannot parse empty config values")
		}

		var configPrefix, configValue string
		values := strings.Split(raw, ",")

		for _, value := range values {
			value = strings.Trim(value, " ")
			config := strings.SplitN(value, ".", 2)
			configPrefix = config[0]
			if len(config) == 2 {
				configValue = config[1]
			} else {
				configValue = configPrefix
			}

			switch configPrefix {
			case configOperator:
				operatorConfig = append(operatorConfig, configValue)
			case configCluster:
				clusterConfig = append(clusterConfig, configValue)
			case configEtcd:
				etcdConfig = append(etcdConfig, configValue)
			default:
				clusterConfig = append(clusterConfig, value)
			}
		}
	}

	if len(operatorConfig) > 0 {
		c.operatorConfig = strings.Join(operatorConfig, ",")
	}

	if len(clusterConfig) > 0 {
		c.clusterConfig = strings.Join(clusterConfig, ",")
	}

	if len(etcdConfig) > 0 {
		c.etcdConfig = strings.Join(etcdConfig, ",")
	}

	return nil
}
