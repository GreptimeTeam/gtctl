package list

import (
	"context"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

func NewListClustersCommand(l log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all GreptimeDB clusters",
		Long:  `List all GreptimeDB clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := manager.New(l, false)
			if err != nil {
				return err
			}

			ctx := context.TODO()
			clusters, err := m.ListClusters(ctx, &manager.ListClusterOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			} else if errors.IsNotFound(err) || (clusters != nil && len(clusters.Items) == 0) {
				l.Infof("clusters not found\n")
				return nil
			}

			// TODO(zyy17): more human friendly output format.
			for _, cluster := range clusters.Items {
				l.Infof("Cluster '%s' in '%s' namespace is running, create at %s\n", cluster.Name, cluster.Namespace, cluster.CreationTimestamp)
			}

			return nil
		},
	}

	return cmd
}
