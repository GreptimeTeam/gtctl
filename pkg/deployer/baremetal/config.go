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
	"net"
	"os"
	"strconv"
)

// Config is the desired state of a GreptimeDB cluster on bare metal.
type Config struct {
	Cluster *Cluster `yaml:"cluster"`
	Etcd    *Etcd    `yaml:"etcd"`
}

type Cluster struct {
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
	Replicas int `yaml:"replicas"`
	NodeID   int `yaml:"nodeID"`

	MySQLAddr string `yaml:"mysqlAddr"`
	RPCAddr   string `yaml:"rpcAddr"`
	HTTPAddr  string `yaml:"httpAddr"`

	DataDir      string `yaml:"dataDir"`
	WalDir       string `yaml:"walDir"`
	ProcedureDir string `yaml:"procedureDir"`
}

type Meta struct {
	StoreAddr  string `yaml:"storeAddr"`
	ServerAddr string `yaml:"serverAddr"`
	BindAddr   string `yaml:"bindAddr"`
	HTTPAddr   string `yaml:"httpAddr"`
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
	if c.Cluster == nil {
		return fmt.Errorf("empty cluster config")
	}
	if c.Etcd == nil {
		return fmt.Errorf("empty etcd config")
	}

	if err := c.Cluster.validate(); err != nil {
		return err
	}
	if err := c.Etcd.validate(); err != nil {
		return err
	}

	return nil
}

func (cluster *Cluster) validate() error {
	if cluster.Artifact == nil {
		return fmt.Errorf("cluster artifact field is nil")
	}
	if err := cluster.Artifact.validate(); err != nil {
		return err
	}

	if cluster.Frontend == nil {
		return fmt.Errorf("frontend field is nil")
	}
	if err := cluster.Frontend.validate(); err != nil {
		return err
	}

	if cluster.Meta == nil {
		return fmt.Errorf("meta field is nil")
	}
	if err := cluster.Meta.validate(); err != nil {
		return err
	}

	if cluster.Datanode == nil {
		return fmt.Errorf("datanode field is nil")
	}
	if err := cluster.Datanode.validate(); err != nil {
		return err
	}

	return nil
}

func (etcd *Etcd) validate() error {
	if etcd.Artifact == nil {
		return fmt.Errorf("etcd artifact field is nil")
	}
	if err := etcd.Artifact.validate(); err != nil {
		return err
	}
	return nil
}

// TODO(zyy17): Add the validation of the options.
func (frontend *Frontend) validate() error {
	return nil
}

func (meta *Meta) validate() error {
	if meta.HTTPAddr == "" {
		return fmt.Errorf("empty meta http addr")
	}
	if err := checkAddr(meta.HTTPAddr); err != nil {
		return err
	}
	return nil
}

func (datanode *Datanode) validate() error {
	if datanode.Replicas <= 0 {
		return fmt.Errorf("invalid replicas '%d'", datanode.Replicas)
	}

	if datanode.NodeID < 0 {
		return fmt.Errorf("invalid nodeID '%d'", datanode.NodeID)
	}

	if datanode.MySQLAddr == "" {
		return fmt.Errorf("empty datanode mysql addr")
	}
	if err := checkAddr(datanode.MySQLAddr); err != nil {
		return err
	}

	if datanode.RPCAddr == "" {
		return fmt.Errorf("empty datanode rpc addr")
	}
	if err := checkAddr(datanode.RPCAddr); err != nil {
		return err
	}

	if datanode.HTTPAddr == "" {
		return fmt.Errorf("empty datanode http addr")
	}
	if err := checkAddr(datanode.HTTPAddr); err != nil {
		return err
	}

	return nil
}

func (artifact *Artifact) validate() error {
	if artifact.Version == "" && artifact.Local == "" {
		return fmt.Errorf("empty artifact")
	}

	if artifact.Local != "" {
		if _, err := os.Stat(artifact.Local); os.IsNotExist(err) {
			return fmt.Errorf("local artifact %s does not exist", artifact.Local)
		}
	}

	return nil
}

func checkAddr(addr string) error {
	addr, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("invalid ip address '%s'", addr)
	}

	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port '%s'", port)
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
				HTTPAddr:   "0.0.0.0:14001",
			},
			Datanode: &Datanode{
				Replicas:  3,
				RPCAddr:   "0.0.0.0:14100",
				MySQLAddr: "0.0.0.0:14200",
				HTTPAddr:  "0.0.0.0:14300",
			},
		},
		Etcd: &Etcd{
			Artifact: &Artifact{
				Version: DefaultEtcdVersion,
			},
		},
	}
}
