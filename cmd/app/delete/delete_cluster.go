package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/log"
)

type deleteOptions struct {
	Namespace    string
	TearDownEtcd bool
}

func NewDeleteClusterCommand(l log.Logger) *cobra.Command {
	var options deleteOptions

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Delete a GreptimeDB cluster.",
		Long:  `Delete a GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			clusterName, namespace := args[0], options.Namespace
			l.Infof("⚠️ Deleting cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(namespace))

			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			ctx := context.TODO()
			_, err = manager.GetCluster(ctx, args[0], options.Namespace)
			if err != nil && errors.IsNotFound(err) {
				l.Infof("Cluster '%s' in '%s' not found\n", clusterName, namespace)
				return nil
			} else if err != nil {
				return err
			}

			if err := manager.DeleteCluster(ctx, args[0], options.Namespace, options.TearDownEtcd); err != nil {
				return err
			}

			// TODO(zyy17): Should we wait until the cluster is actually deleted?
			l.Infof("Cluster '%s' in namespace '%s' is deleted!\n", clusterName, namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.TearDownEtcd, "tear-down-etcd", false, "Tear down etcd cluster.")

	return cmd
}
