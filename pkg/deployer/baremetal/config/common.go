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

package config

const (
	// GtctlDir is the root directory that contains states of cluster info.
	GtctlDir = ".gtctl"

	DefaultEtcdVersion     = "v3.5.7"
	DefaultGreptimeVersion = "latest"
)

// Config is the desired state of a GreptimeDB cluster on bare metal.
//
// The field of Config that with `validate` tag will be validated
// against its requirement. Each filed has only one requirement.
//
// Each field of Config can also have its own exported method `Validate`.
type Config struct {
	Cluster *Cluster `yaml:"cluster" validate:"required"`
	Etcd    *Etcd    `yaml:"etcd" validate:"required"`
}

type Cluster struct {
	Artifact *Artifact `yaml:"artifact" validate:"required"`
	Frontend *Frontend `yaml:"frontend" validate:"required"`
	MetaSrv  *MetaSrv  `yaml:"meta" validate:"required"`
	Datanode *Datanode `yaml:"datanode" validate:"required"`
}

type Artifact struct {
	// Local is the local path of binary(greptime or etcd).
	Local string `yaml:"local" validate:"omitempty,file"`

	// Version is the release version of binary(greptime or etcd).
	// Usually, it points to the version of binary of GitHub release.
	Version string `yaml:"version"`
}

func DefaultConfig() *Config {
	return &Config{
		Cluster: &Cluster{
			Artifact: &Artifact{
				Version: DefaultGreptimeVersion,
			},
			Frontend: &Frontend{},
			MetaSrv: &MetaSrv{
				StoreAddr:  "127.0.0.1:2379",
				ServerAddr: "0.0.0.0:3002",
				HTTPAddr:   "0.0.0.0:14001",
			},
			Datanode: &Datanode{
				Replicas: 3,
				RPCAddr:  "0.0.0.0:14100",
				HTTPAddr: "0.0.0.0:14300",
			},
		},
		Etcd: &Etcd{
			Artifact: &Artifact{
				Version: DefaultEtcdVersion,
			},
		},
	}
}
