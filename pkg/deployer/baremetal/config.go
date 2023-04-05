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
)

// Config is the desired state of a GreptimeDB cluster on bare metal.
type Config struct {
	Cluster *Cluster `yaml:"cluster"`
	Etcd    *Etcd    `yaml:"etcd"`
}

type Cluster struct {
	Name     string    `yaml:"name"`
	Artifact *Artifact `yaml:"artifact"`
	Frontend *Frontend `yaml:"frontend"`
	Meta     *Meta     `yaml:"meta"`
	Datanode *Datanode `yaml:"datanode"`
}

type Frontend struct {
	GRPCAddr     string `yaml:"grpcAddr"`
	HTTPAddr     string `yaml:"httpAddr"`
	PostgresAddr string `yaml:"postgresAddr"`
	MetaAddr     string `yaml:"metaAddr"`
}

type Datanode struct {
	Replicas     int    `yaml:"replicas"`
	NodeID       int    `yaml:"nodeID"`
	MySQLPort    int    `yaml:"mysqlPort"`
	RPCPort      int    `yaml:"rpcPort"`
	DataDir      string `yaml:"dataDir"`
	WalDir       string `yaml:"walDir"`
	ProcedureDir string `yaml:"procedureDir"`
}

type Meta struct {
	StoreAddr  string `yaml:"storeAddr"`
	ServerAddr string `yaml:"serverAddr"`
	BindAddr   string `yaml:"bindAddr"`
}

type Etcd struct {
	Artifact *Artifact `yaml:"artifact"`
}

type Artifact struct {
	// Local is the local path of binary(greptime or etcd).
	Local string `yaml:"local"`

	// Version is the release version of binary(greptime or etcd).
	// Usually, it points to the version of binary of GitHub release.
	Version string `yaml:"version"`
}

func (c *Config) Validate() error {
	// TODO(zyy17): Add the validation of the options.
	if c.Cluster == nil || c.Cluster.Artifact == nil {
		return fmt.Errorf("invalid cluster config")
	}

	if c.Etcd == nil || c.Etcd.Artifact == nil {
		return fmt.Errorf("invalid etcd config")
	}

	if c.Cluster.Artifact.Version == "" && c.Cluster.Artifact.Local == "" {
		return fmt.Errorf("empty artifact")
	}

	if c.Cluster.Artifact.Local != "" {
		if _, err := os.Stat(c.Cluster.Artifact.Local); os.IsNotExist(err) {
			return fmt.Errorf("local artifact %s does not exist", c.Cluster.Artifact.Local)
		}
	}

	if c.Etcd.Artifact.Local != "" {
		if _, err := os.Stat(c.Etcd.Artifact.Local); os.IsNotExist(err) {
			return fmt.Errorf("local artifact %s does not exist", c.Etcd.Artifact.Local)
		}
	}

	if c.Cluster.Datanode == nil {
		return fmt.Errorf("invalid datanode")
	}

	if c.Cluster.Datanode.Replicas <= 0 {
		return fmt.Errorf("invalid replicas '%d'", c.Cluster.Datanode.Replicas)
	}

	return nil
}

func defaultConfig() *Config {
	return &Config{
		Cluster: &Cluster{
			Artifact: &Artifact{
				Version: DefaultGreptimeVersion,
			},
			Frontend: &Frontend{},
			Meta: &Meta{
				StoreAddr:  "127.0.0.1:2379",
				ServerAddr: "0.0.0.0:3002",
			},
			Datanode: &Datanode{
				Replicas:  3,
				RPCPort:   14100,
				MySQLPort: 14200,
			},
		},
		Etcd: &Etcd{
			Artifact: &Artifact{
				Version: DefaultEtcdVersion,
			},
		},
	}
}
