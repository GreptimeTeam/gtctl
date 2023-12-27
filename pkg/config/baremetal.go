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

import (
	"time"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
)

// BareMetalClusterMetadata stores metadata of a GreptimeDB cluster.
type BareMetalClusterMetadata struct {
	Config        *BareMetalClusterConfig `yaml:"config"`
	CreationDate  time.Time               `yaml:"creationDate"`
	ClusterDir    string                  `yaml:"clusterDir"`
	ForegroundPid int                     `yaml:"foregroundPid"`
}

// BareMetalClusterConfig is the desired state of a GreptimeDB cluster on bare metal.
//
// The field of BareMetalClusterConfig that with `validate` tag will be validated
// against its requirement. Each filed has only one requirement.
//
// Each field of BareMetalClusterConfig can also have its own exported method `Validate`.
type BareMetalClusterConfig struct {
	Cluster *BareMetalClusterComponentsConfig `yaml:"cluster" validate:"required"`
	Etcd    *Etcd                             `yaml:"etcd" validate:"required"`
}

type BareMetalClusterComponentsConfig struct {
	Artifact *Artifact `yaml:"artifact" validate:"required"`
	Frontend *Frontend `yaml:"frontend" validate:"required"`
	MetaSrv  *MetaSrv  `yaml:"meta" validate:"required"`
	Datanode *Datanode `yaml:"datanode" validate:"required"`
}

type Artifact struct {
	// Local is the local path of binary(greptime or etcd).
	Local string `yaml:"local" validate:"omitempty,filepath"`

	// Version is the release version of binary(greptime or etcd).
	// Usually, it points to the version of binary of GitHub release.
	Version string `yaml:"version"`
}

type Datanode struct {
	NodeID       int    `yaml:"nodeID" validate:"gte=0"`
	RPCAddr      string `yaml:"rpcAddr" validate:"required,hostname_port"`
	HTTPAddr     string `yaml:"httpAddr" validate:"required,hostname_port"`
	DataDir      string `yaml:"dataDir" validate:"omitempty,dirpath"`
	WalDir       string `yaml:"walDir" validate:"omitempty,dirpath"`
	ProcedureDir string `yaml:"procedureDir" validate:"omitempty,dirpath"`

	Replicas int    `yaml:"replicas" validate:"gt=0"`
	Config   string `yaml:"config" validate:"omitempty,filepath"`
	LogLevel string `yaml:"logLevel"`
}

type Frontend struct {
	GRPCAddr     string `yaml:"grpcAddr" validate:"omitempty,hostname_port"`
	HTTPAddr     string `yaml:"httpAddr" validate:"omitempty,hostname_port"`
	PostgresAddr string `yaml:"postgresAddr" validate:"omitempty,hostname_port"`
	MetaAddr     string `yaml:"metaAddr" validate:"omitempty,hostname_port"`
	MysqlAddr    string `yaml:"mysqlAddr" validate:"omitempty,hostname_port"`
	OpentsdbAddr string `yaml:"opentsdbAddr" validate:"omitempty,hostname_port"`

	Replicas int    `yaml:"replicas" validate:"gt=0"`
	Config   string `yaml:"config" validate:"omitempty,filepath"`
	LogLevel string `yaml:"logLevel"`
}

type MetaSrv struct {
	StoreAddr  string `yaml:"storeAddr" validate:"hostname_port"`
	ServerAddr string `yaml:"serverAddr" validate:"hostname_port"`
	BindAddr   string `yaml:"bindAddr" validate:"omitempty,hostname_port"`
	HTTPAddr   string `yaml:"httpAddr" validate:"required,hostname_port"`

	Replicas int    `yaml:"replicas" validate:"gt=0"`
	Config   string `yaml:"config" validate:"omitempty,filepath"`
	LogLevel string `yaml:"logLevel"`
}

type Etcd struct {
	Artifact *Artifact `yaml:"artifact" validate:"required"`
}

func DefaultBareMetalConfig() *BareMetalClusterConfig {
	return &BareMetalClusterConfig{
		Cluster: &BareMetalClusterComponentsConfig{
			Artifact: &Artifact{
				Version: artifacts.LatestVersionTag,
			},
			Frontend: &Frontend{
				Replicas: 1,
			},
			MetaSrv: &MetaSrv{
				Replicas:   1,
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
				Version: artifacts.DefaultEtcdBinVersion,
			},
		},
	}
}
