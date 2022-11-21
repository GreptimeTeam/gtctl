package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type deleteClusterCliOptions struct {
	Namespace     string
	TearDownEtcd  bool
	ETCDNamespace string
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
			clusterExist := true
			l.Infof("⚠️ Deleting cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(namespace))

			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			_, err = m.GetCluster(ctx, &manager.GetClusterOptions{
				ClusterName: clusterName,
				Namespace:   options.Namespace,
			})
			if errors.IsNotFound(err) {
				l.Infof("Cluster '%s' in '%s' not found\n", clusterName, namespace)
				clusterExist = false
			} else if err != nil && !errors.IsNotFound(err) {
				return err
			}

			if clusterExist {
				if err := m.DeleteCluster(ctx, &manager.DeleteClusterOption{
					ClusterName: clusterName,
					Namespace:   options.Namespace,
				}); err != nil {
					return err
				}

				// TODO(zyy17): Should we wait until the cluster is actually deleted?
				l.Infof("Cluster '%s' in namespace '%s' is deleted!\n", clusterName, namespace)
			}

			if options.TearDownEtcd {
				l.Infof("⚠️ Deleting etcd cluster in namespace '%s'...\n", log.Bold(options.ETCDNamespace))
				if err := m.DeleteETCDCluster(ctx, &manager.DeleteETCDClusterOption{
					Namespace: options.ETCDNamespace,
				}); err != nil {
					return err
				}
				l.Infof("ETCD cluster in namespace '%s' is deleted!\n", options.ETCDNamespace)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")
	cmd.Flags().StringVar(&options.ETCDNamespace, "etcd-namespace", "default", "Namespace of etcd cluster.")

	return cmd
}
