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
	"reflect"
	"strconv"
	"strings"
)

const (
	ValidationTag = "validate"

	validateNotNil   = "not-nil"
	validateNotEmpty = "not-empty"
	validateIsAddr   = "is-addr"
)

// Config is the desired state of a GreptimeDB cluster on bare metal.
//
// The field of Config that with `validate` tag will be validated
// against its requirement. Each filed has only one requirement.
//
// Each field of Config can also have its own exported method `Validate`.
type Config struct {
	Cluster *Cluster `yaml:"cluster" validate:"not-nil"`
	Etcd    *Etcd    `yaml:"etcd" validate:"not-nil"`
}

type Cluster struct {
	Artifact *Artifact `yaml:"artifact" validate:"not-nil"`
	Frontend *Frontend `yaml:"frontend" validate:"not-nil"`
	Meta     *Meta     `yaml:"meta" validate:"not-nil"`
	Datanode *Datanode `yaml:"datanode" validate:"not-nil"`
}

type Frontend struct {
	GRPCAddr     string `yaml:"grpcAddr" validate:"is-addr"`
	HTTPAddr     string `yaml:"httpAddr" validate:"is-addr"`
	PostgresAddr string `yaml:"postgresAddr" validate:"is-addr"`
	MetaAddr     string `yaml:"metaAddr" validate:"is-addr"`

	LogLevel string `yaml:"logLevel"`
}

type Datanode struct {
	Replicas int `yaml:"replicas"`
	NodeID   int `yaml:"nodeID"`

	RPCAddr  string `yaml:"rpcAddr" validate:"not-empty,is-addr"`
	HTTPAddr string `yaml:"httpAddr" validate:"not-empty,is-addr"`

	DataDir      string `yaml:"dataDir"`
	WalDir       string `yaml:"walDir"`
	ProcedureDir string `yaml:"procedureDir"`

	LogLevel string `yaml:"logLevel"`
}

type Meta struct {
	StoreAddr  string `yaml:"storeAddr" validate:"is-addr"`
	ServerAddr string `yaml:"serverAddr" validate:"is-addr"`
	BindAddr   string `yaml:"bindAddr" validate:"is-addr"`
	HTTPAddr   string `yaml:"httpAddr" validate:"not-empty,is-addr"`

	LogLevel string `yaml:"logLevel"`
}

type Etcd struct {
	Artifact *Artifact `yaml:"artifact" validate:"not-nil"`
}

type Artifact struct {
	// Local is the local path of binary(greptime or etcd).
	Local string `yaml:"local"`

	// Version is the release version of binary(greptime or etcd).
	// Usually, it points to the version of binary of GitHub release.
	Version string `yaml:"version"`
}

// ValidateConfig validate config in bare-metal mode.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("no config to validate")
	}

	err := validateConfigWithSingleValue(config, "")
	if err != nil {
		return err
	}

	return nil
}

// validateConfigWithSingleValue validate every single config value.
func validateConfigWithSingleValue(config interface{}, path string) error {
	valueOf := reflect.ValueOf(config)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}
	if valueOf.Kind() != reflect.Struct {
		return nil
	}

	typeOf := reflect.TypeOf(config)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	for i := 0; i < valueOf.NumField(); i++ {
		validateTypes := typeOf.Field(i).Tag.Get(ValidationTag)

		fieldPath := fmt.Sprintf("%s.%s", path, typeOf.Field(i).Name)
		if len(validateTypes) > 0 {
			if err := validateTags(validateTypes, valueOf.Field(i)); err != nil {
				return fmt.Errorf("error at field `%s`: %v", fieldPath, err)
			}
		}
		// Perform field validation that defined by `Validate` method.
		if method := valueOf.Field(i).MethodByName("Validate"); method.IsValid() {
			if err := method.Call(nil)[0]; !err.IsNil() {
				return fmt.Errorf("error at field `%s`: %v", fieldPath, err)
			}
		}

		if err := validateConfigWithSingleValue(valueOf.Field(i).Interface(), fieldPath); err != nil {
			return err
		}
	}
	return nil
}

func validateTags(types string, value reflect.Value) error {
	tags := strings.Split(types, ",")
	for _, tag := range tags {
		switch tag {
		case validateNotNil:
			if value.Type().Kind() == reflect.Ptr && value.IsNil() {
				return fmt.Errorf("got nil")
			}
		case validateNotEmpty:
			if value.Type().Kind() != reflect.Ptr && value.Len() == 0 {
				return fmt.Errorf("got empty")
			}
		case validateIsAddr:
			if value.Type().Kind() == reflect.String && value.Len() > 0 {
				return validateAddr(value.String())
			}
		default:
			return fmt.Errorf("unfamiliar validation tag: %s", tag)
		}
	}
	return nil
}

func validateAddr(addr string) error {
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

func (datanode *Datanode) Validate() error {
	if datanode.Replicas <= 0 {
		return fmt.Errorf("invalid replicas '%d'", datanode.Replicas)
	}

	if datanode.NodeID < 0 {
		return fmt.Errorf("invalid nodeID '%d'", datanode.NodeID)
	}
	return nil
}

func (artifact *Artifact) Validate() error {
	if artifact.Version == "" && artifact.Local == "" {
		return fmt.Errorf("empty artifact")
	}

	if artifact.Local != "" {
		fileinfo, err := os.Stat(artifact.Local)
		if os.IsNotExist(err) {
			return fmt.Errorf("artifact '%s' not exist", artifact.Local)
		}
		if fileinfo.IsDir() {
			return fmt.Errorf("artifact '%s' should be file, not directory", artifact.Local)
		}
		if err != nil {
			return err
		}
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
