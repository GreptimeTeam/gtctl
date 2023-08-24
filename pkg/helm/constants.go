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

package helm

const (
	DefaultChartsCache = ".gtctl/charts-cache"

	GreptimeChartIndexURL           = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"
	GreptimeChartReleaseDownloadURL = "https://github.com/GreptimeTeam/helm-charts/releases/download"

	GreptimeDBChartName         = "greptimedb"
	GreptimeDBOperatorChartName = "greptimedb-operator"
	EtcdBitnamiOCIRegistry      = "oci://registry-1.docker.io/bitnamicharts/etcd"
	DefaultEtcdChartVersion     = "9.2.0"
)
