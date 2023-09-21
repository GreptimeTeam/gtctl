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

type Datanode struct {
	Replicas int `yaml:"replicas" validate:"gt=0"`
	NodeID   int `yaml:"nodeID" validate:"gte=0"`

	RPCAddr  string `yaml:"rpcAddr" validate:"required,hostname_port"`
	HTTPAddr string `yaml:"httpAddr" validate:"required,hostname_port"`

	DataDir      string `yaml:"dataDir" validate:"omitempty,dirpath"`
	WalDir       string `yaml:"walDir" validate:"omitempty,dirpath"`
	ProcedureDir string `yaml:"procedureDir" validate:"omitempty,dirpath"`
	Config       string `yaml:"config" validate:"omitempty,filepath"`

	LogLevel string `yaml:"logLevel"`
}
