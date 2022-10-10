package get

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

type getOptions struct {
	Namespace string
}

func NewGetClusterCommand() *cobra.Command {
	var options getOptions
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Get a GreptimeDB cluster.",
		Long:  `Get a GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}
			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			ctx := context.TODO()
			gtCluster, err := manager.GetCluster(ctx, args[0], options.Namespace)
			if err != nil && errors.IsNotFound(err) {
				log.Printf("cluster %s in %s not found\n", args[0], options.Namespace)
				return nil
			} else if err != nil {
				return err
			}

			log.Printf("Cluster '%s' in '%s' namespace is running, create at %s\n", gtCluster.Name, gtCluster.Namespace, gtCluster.CreationTimestamp)
			return nil
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")

	return cmd
}
