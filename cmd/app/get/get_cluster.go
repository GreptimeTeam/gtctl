package get

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

type getOptions struct {
	ClusterName string
	Namespace   string
}

func NewGetClusterCommand() *cobra.Command {
	var options getOptions

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Get a GreptimeDB cluster.",
		Long:  `Get a GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			ctx := context.TODO()
			gtCluster, err := manager.GetCluster(ctx, options.ClusterName, options.Namespace)
			if err != nil && errors.IsNotFound(err) {
				log.Printf("cluster %s in %s not found\n", options.ClusterName, options.Namespace)
				return nil
			} else if err != nil {
				return err
			}

			log.Printf("Cluster '%s' in '%s' namespace is running, create at %s\n", gtCluster.Name, gtCluster.Namespace, gtCluster.CreationTimestamp)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.ClusterName, "name", "greptimedb", "Name of GreptimeDB cluster.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")

	return cmd
}
