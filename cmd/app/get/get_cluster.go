package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type getClusterCliOptions struct {
	Namespace string
}

func NewGetClusterCommand(l log.Logger) *cobra.Command {
	var options getClusterCliOptions
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Get a GreptimeDB cluster.",
		Long:  `Get a GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			var (
				ctx         = context.TODO()
				clusterName = args[0]
				namespace   = options.Namespace
			)

			cluster, err := m.GetCluster(ctx, &manager.GetClusterOptions{
				ClusterName: clusterName,
				Namespace:   namespace,
			})
			if err != nil && errors.IsNotFound(err) {
				l.Infof("cluster %s in %s not found\n", clusterName, namespace)
				return nil
			} else if err != nil {
				return err
			}

			l.Infof("Cluster '%s' in '%s' namespace is running, create at %s\n", cluster.Name, cluster.Namespace, cluster.CreationTimestamp)
			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")

	return cmd
}
