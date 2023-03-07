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

const (
	// GtctlDir is the root directory that contains states of cluster info.
	GtctlDir = ".gtctl"

	// PackagesDir will store the downloaded binary packages.
	PackagesDir = "packages"

	// BinaryDir will store the uncompressed and executable binary.
	BinaryDir = "bin"

	// LogsDir will store the logging from multiple components.
	LogsDir = "logs"

	// PidsDir will store the pid of multiple components.
	PidsDir = "pids"

	// DataDir will store the data of cluster, incluing metadata and data.
	DataDir = "data"

	DefaultEtcdVersion     = "v3.5.7"
	DefaultGreptimeVersion = "v0.1.2"

	EtcdBinaryDownloadURLPrefix = "https://github.com/etcd-io/etcd/releases/download/v3.5.7/etcd-v3.5.7-"

	GreptimeBinaryDownloadURLPrefix = "https://github.com/GreptimeTeam/greptimedb/releases/download"
)
