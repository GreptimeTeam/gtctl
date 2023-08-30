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

package get

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/component"
	bmconfig "github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/deployer/k8s"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type getClusterCliOptions struct {
	Namespace string

	// The options for getting GreptimeDBCluster in bare-metal.
	BareMetal bool
}

func NewGetClusterCommand(l logger.Logger) *cobra.Command {
	var options getClusterCliOptions

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get GreptimeDB cluster",
		Long:  `Get GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
			)

			nn := types.NamespacedName{
				Namespace: options.Namespace,
				Name:      clusterName,
			}

			if options.BareMetal {
				return getClusterFromBareMetal(ctx, l, nn, table)
			} else {
				return getClusterFromKubernetes(ctx, l, nn, table)
			}
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.BareMetal, "bare-metal", false, "Get the greptimedb cluster on bare-metal environment.")

	return cmd
}

func getClusterFromKubernetes(ctx context.Context, l logger.Logger, nn types.NamespacedName, table *tablewriter.Table) error {
	deployer, err := k8s.NewDeployer(l)
	if err != nil {
		return err
	}

	cluster, err := deployer.GetGreptimeDBCluster(ctx, nn.String(), nil)
	if err != nil && errors.IsNotFound(err) {
		l.Errorf("cluster %s in %s not found\n", nn.Name, nn.Namespace)
		return nil
	}
	if err != nil {
		return err
	}

	rawCluster, ok := cluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
	if !ok {
		return fmt.Errorf("invalid cluster type")
	}

	l.V(0).Infof("Cluster '%s' in '%s' namespace is running, create at %s\n",
		rawCluster.Name, rawCluster.Namespace, rawCluster.CreationTimestamp)
	return nil
}

func getClusterFromBareMetal(ctx context.Context, l logger.Logger, nn types.NamespacedName, table *tablewriter.Table) error {
	deployer, err := baremetal.NewDeployer(l, nn.Name, baremetal.WithCreateNoDirs())
	if err != nil {
		return nil
	}

	cluster, err := deployer.GetGreptimeDBCluster(ctx, nn.Name, nil)
	if err != nil {
		return err
	}

	rawCluster, ok := cluster.Raw.(*bmconfig.MetaConfig)
	if !ok {
		return fmt.Errorf("invalid cluster type")
	}

	headers, footers, bulk := collectClusterInfoFromBareMetal(rawCluster)
	table.SetHeader(headers)
	table.AppendBulk(bulk)
	table.Render()

	for _, footer := range footers {
		l.V(0).Info(footer)
	}

	return nil
}

func collectClusterInfoFromBareMetal(data *bmconfig.MetaConfig) (
	headers, footers []string, bulk [][]string) {
	headers = []string{"COMPONENT", "PID"}

	pidsDir := path.Join(data.ClusterDir, bmconfig.PidsDir)
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

	rows(component.Frontend, data.Cluster.Frontend.Replicas)
	rows(component.DataNode, data.Cluster.Datanode.Replicas)
	rows(component.MetaSrv, data.Cluster.MetaSrv.Replicas)

	bulk = append(bulk, []string{component.Etcd, pidsMap[component.Etcd]})

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
			if d.Name() == bmconfig.PidsDir {
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
