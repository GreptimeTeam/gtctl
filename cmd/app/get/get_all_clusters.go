package get

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

func NewGetAllClustersCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Get all GreptimeDB clusters.",
		Long:  `Get all GreptimeDB clusters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}
			ctx := context.TODO()
			gtClusters, err := manager.GetAllClusters(ctx)
			if err != nil && !errors.IsNotFound(err) {
				return err
			} else if errors.IsNotFound(err) || (gtClusters != nil && len(gtClusters.Items) == 0) {
				log.Printf("clusters not found")
				return nil
			}

			for _, gtCluster := range gtClusters.Items {
				log.Printf("Cluster '%s' in '%s' namespace is running, create at %s\n", gtCluster.Name, gtCluster.Namespace, gtCluster.CreationTimestamp)
			}
			return nil
		},
	}

	return cmd
}
