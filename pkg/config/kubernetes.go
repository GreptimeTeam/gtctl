// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

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

type SetValues struct {
	RawConfig []string

	OperatorConfig string
	ClusterConfig  string
	EtcdConfig     string
}

// Parse parses raw config values and classify it to different
// categories of config type by its prefix.
func (c *SetValues) Parse() error {
	var (
		operatorConfig []string
		clusterConfig  []string
		etcdConfig     []string
	)

	for _, raw := range c.RawConfig {
		if len(raw) == 0 {
			return fmt.Errorf("cannot parse empty config values")
		}

		var configPrefix, configValue string
		values := strings.Split(raw, ",")

		for _, value := range values {
			value = strings.Trim(value, " ")
			cfg := strings.SplitN(value, ".", 2)
			configPrefix = cfg[0]
			if len(cfg) == 2 {
				configValue = cfg[1]
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
		c.OperatorConfig = strings.Join(operatorConfig, ",")
	}
	if len(clusterConfig) > 0 {
		c.ClusterConfig = strings.Join(clusterConfig, ",")
	}
	if len(etcdConfig) > 0 {
		c.EtcdConfig = strings.Join(etcdConfig, ",")
	}

	return nil
}
