package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type deleteClusterCliOptions struct {
	Namespace    string
	TearDownEtcd bool
}

func NewDeleteClusterCommand(l log.Logger) *cobra.Command {
	var options deleteClusterCliOptions

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a GreptimeDB cluster",
		Long:  `Delete a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			clusterName, namespace := args[0], options.Namespace
			l.Infof("⚠️ Deleting cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(namespace))

			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			cluster, err := m.GetCluster(ctx, &manager.GetClusterOptions{
				ClusterName: clusterName,
				Namespace:   options.Namespace,
			})
			if errors.IsNotFound(err) {
				l.Infof("Cluster '%s' in '%s' not found\n", clusterName, namespace)
				return nil
			} else if err != nil && !errors.IsNotFound(err) {
				return err
			}

			etcdNamespace := strings.Split(strings.Split(cluster.Spec.Meta.EtcdEndpoints[0], ".")[1], ":")[0]
			if err := m.DeleteCluster(ctx, &manager.DeleteClusterOption{
				ClusterName: clusterName,
				Namespace:   options.Namespace,
			}); err != nil {
				return err
			}

			// TODO(zyy17): Should we wait until the cluster is actually deleted?
			l.Infof("Cluster '%s' in namespace '%s' is deleted!\n", clusterName, namespace)

			if options.TearDownEtcd {
				l.Infof("⚠️ Deleting etcd cluster in namespace '%s'...\n", log.Bold(etcdNamespace))
				if err := m.DeleteEtcdCluster(ctx, &manager.DeleteEtcdClusterOption{
					Name:      clusterName + "-etcd",
					Namespace: etcdNamespace,
				}); err != nil {
					return err
				}
				l.Infof("Etcd cluster in namespace '%s' is deleted!\n", etcdNamespace)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")

	return cmd
}
