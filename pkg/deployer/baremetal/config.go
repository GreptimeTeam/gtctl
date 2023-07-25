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

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

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
	Meta     *Meta     `yaml:"meta" validate:"required"`
	Datanode *Datanode `yaml:"datanode" validate:"required"`
}

type Frontend struct {
	GRPCAddr     string `yaml:"grpcAddr" validate:"omitempty,hostname_port"`
	HTTPAddr     string `yaml:"httpAddr" validate:"omitempty,hostname_port"`
	PostgresAddr string `yaml:"postgresAddr" validate:"omitempty,hostname_port"`
	MetaAddr     string `yaml:"metaAddr" validate:"omitempty,hostname_port"`

	LogLevel string `yaml:"logLevel"`
}

type Datanode struct {
	Replicas int `yaml:"replicas" validate:"gt=0"`
	NodeID   int `yaml:"nodeID" validate:"gte=0"`

	RPCAddr  string `yaml:"rpcAddr" validate:"required,hostname_port"`
	HTTPAddr string `yaml:"httpAddr" validate:"required,hostname_port"`

	DataDir      string `yaml:"dataDir" validate:"omitempty,dirpath"`
	WalDir       string `yaml:"walDir" validate:"omitempty,dirpath"`
	ProcedureDir string `yaml:"procedureDir" validate:"omitempty,dirpath"`

	LogLevel string `yaml:"logLevel"`
}

type Meta struct {
	StoreAddr  string `yaml:"storeAddr" validate:"hostname_port"`
	ServerAddr string `yaml:"serverAddr" validate:"hostname_port"`
	BindAddr   string `yaml:"bindAddr" validate:"omitempty,hostname_port"`
	HTTPAddr   string `yaml:"httpAddr" validate:"required,hostname_port"`

	LogLevel string `yaml:"logLevel"`
}

type Etcd struct {
	Artifact *Artifact `yaml:"artifact" validate:"required"`
}

type Artifact struct {
	// Local is the local path of binary(greptime or etcd).
	Local string `yaml:"local" validate:"omitempty,file"`

	// Version is the release version of binary(greptime or etcd).
	// Usually, it points to the version of binary of GitHub release.
	Version string `yaml:"version"`
}

// ValidateConfig validate config in bare-metal mode.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("no config to validate")
	}

	validate = validator.New()

	// Register custom validation method for Artifact.
	validate.RegisterStructValidation(ValidateArtifact, Artifact{})

	err := validate.Struct(config)
	if err != nil {
		return err
	}

	return nil
}

func ValidateArtifact(sl validator.StructLevel) {
	artifact := sl.Current().Interface().(Artifact)
	if len(artifact.Version) == 0 && len(artifact.Local) == 0 {
		sl.ReportError(sl.Current().Interface(), "Artifact", "Version/Local", "", "")
	}
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
