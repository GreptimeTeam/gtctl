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

package artifacts

const (
	// GreptimeChartIndexURL is the URL of the Greptime chart index.
	GreptimeChartIndexURL = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"

	// GreptimeChartReleaseDownloadURL is the URL of the Greptime charts that stored in the GitHub release.
	GreptimeChartReleaseDownloadURL = "https://github.com/GreptimeTeam/helm-charts/releases/download"

	// GreptimeCNCharts is the URL of the Greptime charts that stored in the S3 bucket of the CN region.
	GreptimeCNCharts = "https://downloads.greptime.cn/releases/charts"

	// GreptimeDBCNBinaries is the URL of the GreptimeDB binaries that stored in the S3 bucket of the CN region.
	GreptimeDBCNBinaries = "https://downloads.greptime.cn/releases/greptimedb"

	// EtcdCNBinaries is the URL of the etcd binaries that stored in the S3 bucket of the CN region.
	EtcdCNBinaries = "https://downloads.greptime.cn/releases/etcd"

	// LatestVersionTag is the tag of the latest version.
	LatestVersionTag = "latest"

	// EtcdOCIRegistry is the OCI registry of the etcd chart.
	EtcdOCIRegistry = "oci://registry-1.docker.io/bitnamicharts/etcd"

	// GreptimeGitHubOrg is the GitHub organization of Greptime.
	GreptimeGitHubOrg = "GreptimeTeam"

	// GreptimeDBGithubRepo is the GitHub repository of GreptimeDB.
	GreptimeDBGithubRepo = "greptimedb"

	// EtcdGitHubOrg is the GitHub organization of etcd.
	EtcdGitHubOrg = "etcd-io"

	// EtcdGithubRepo is the GitHub repository of etcd.
	EtcdGithubRepo = "etcd"

	// GreptimeBinName is the artifact name of greptime.
	GreptimeBinName = "greptime"

	// EtcdBinName is the artifact name of etcd.
	EtcdBinName = "etcd"

	// GreptimeDBChartName is the chart name of GreptimeDB.
	GreptimeDBChartName = "greptimedb"

	// GreptimeDBOperatorChartName is the chart name of GreptimeDB operator.
	GreptimeDBOperatorChartName = "greptimedb-operator"

	// EtcdChartName is the chart name of etcd.
	EtcdChartName = "etcd"

	// DefaultEtcdChartVersion is the default etcd chart version.
	DefaultEtcdChartVersion = "9.2.0"

	// DefaultEtcdBinVersion is the default etcd binary version.
	DefaultEtcdBinVersion = "v3.5.7"
)
