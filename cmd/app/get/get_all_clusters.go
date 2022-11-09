package get

import (
	"context"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/log"
)

func NewGetAllClustersCommand(l log.Logger) *cobra.Command {
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
				l.Infof("clusters not found\n")
				return nil
			}

			for _, gtCluster := range gtClusters.Items {
				l.Infof("Cluster '%s' in '%s' namespace is running, create at %s\n", gtCluster.Name, gtCluster.Namespace, gtCluster.CreationTimestamp)
			}
			return nil
		},
	}

	return cmd
}
