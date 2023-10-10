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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"

	"github.com/GreptimeTeam/gtctl/pkg/api/query"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

var _ query.Getter = &Cluster{}

func (c *Cluster) Get(ctx context.Context, options *query.Options) error {
	cluster, err := c.get(ctx, options)
	if err != nil {
		return err
	}

	c.renderGetView(options.Table, cluster)

	return nil
}

func (c *Cluster) get(_ context.Context, options *query.Options) (*ClusterMetadata, error) {
	_, err := os.Stat(c.clusterDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cluster %s is not exist", options.Name)
	}
	if err != nil {
		return nil, err
	}

	ok, err := fileutils.IsFileExists(c.clusterConfigPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("cluster %s is not exist", options.Name)
	}

	var cluster ClusterMetadata
	in, err := os.ReadFile(c.clusterConfigPath)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(in, &cluster); err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (c *Cluster) configGetView(table *tablewriter.Table) {
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
}

func (c *Cluster) renderGetView(table *tablewriter.Table, data *ClusterMetadata) {
	c.configGetView(table)

	headers, footers, bulk := collectClusterInfoFromBareMetal(data)
	table.SetHeader(headers)
	table.AppendBulk(bulk)
	table.Render()

	for _, footer := range footers {
		c.logger.V(0).Info(footer)
	}
}

func collectClusterInfoFromBareMetal(data *ClusterMetadata) (
	headers, footers []string, bulk [][]string) {
	headers = []string{"COMPONENT", "PID"}

	pidsDir := path.Join(data.ClusterDir, PidsDir)
	pidsMap := collectPidsForBareMetal(pidsDir)

	var (
		date = data.CreationDate.String()
		rows = func(name string, replicas int) {
			for i := 0; i < replicas; i++ {
				key := fmt.Sprintf("%s.%d", name, i)
				pid := "N/A"
				if val, ok := pidsMap[key]; ok {
					pid = fmt.Sprintf(".%d: %s", i, val)
				}
				bulk = append(bulk, []string{name, pid})
			}
		}
	)

	rows(string(greptimedbclusterv1alpha1.FrontendComponentKind), data.Cluster.Frontend.Replicas)
	rows(string(greptimedbclusterv1alpha1.DatanodeComponentKind), data.Cluster.Datanode.Replicas)
	rows(string(greptimedbclusterv1alpha1.MetaComponentKind), data.Cluster.MetaSrv.Replicas)

	// TODO(sh2): make "etcd" a const
	bulk = append(bulk, []string{"etcd", pidsMap["etcd"]})

	config, err := yaml.Marshal(data.Config)
	footers = []string{
		fmt.Sprintf("CREATION-DATE: %s", date),
		fmt.Sprintf("GREPTIMEDB-VERSION: %s", data.Cluster.Artifact.Version),
		fmt.Sprintf("ETCD-VERSION: %s", data.Etcd.Artifact.Version),
		fmt.Sprintf("CLUSTER-DIR: %s", data.ClusterDir),
	}
	if err != nil {
		footers = append(footers, fmt.Sprintf("CLUSTER-CONFIG: error retrieving cluster config: %v", err))
	} else {
		footers = append(footers, fmt.Sprintf("CLUSTER-CONFIG:\n%s", string(config)))
	}

	return headers, footers, bulk
}

// collectPidsForBareMetal returns the pid of each component.
func collectPidsForBareMetal(pidsDir string) map[string]string {
	ret := make(map[string]string)

	if err := filepath.WalkDir(pidsDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if d.Name() == PidsDir {
				return nil
			}

			pidPath := filepath.Join(path, "pid")
			pid, err := os.ReadFile(pidPath)
			if err != nil {
				return err
			}

			ret[d.Name()] = string(pid)
		}
		return nil
	}); err != nil {
		return ret
	}

	return ret
}
