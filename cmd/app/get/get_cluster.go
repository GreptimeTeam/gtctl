package get

import (
	"context"
	"log"

	"github.com/spf13/cobra"

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
			if err != nil {
				return err
			}

			log.Printf("Cluster '%s' in '%s' namespace is running, create at %s\n", gtCluster.Name, gtCluster.Namespace, gtCluster.CreationTimestamp)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ClusterName, "cluster-name", "n", "greptimedb", "Name of GreptimeDB cluster.")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "default", "Namespace of GreptimeDB cluster.")

	return cmd
}
