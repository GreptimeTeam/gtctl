package scale

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

type scaleOptions struct {
	ClusterName   string
	Namespace     string
	ComponentType string
	Replicas      int32
}

// FIXME(zyy17): ComponentType should be defined in CRDs.

type ComponentType string

const (
	Frontend ComponentType = "frontend"
	Datanode ComponentType = "datanode"
)

func NewScaleClusterCommand() *cobra.Command {
	var options scaleOptions

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Scale GreptimeDB cluster.",
		Long:  `Scale GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.ComponentType == "" {
				return fmt.Errorf("component type is required")
			}

			if options.ComponentType != string(Frontend) && options.ComponentType != string(Datanode) {
				return fmt.Errorf("component type is invalid")
			}

			if options.Replicas == 0 {
				return fmt.Errorf("replicas should be greater than 0")
			}

			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			ctx := context.TODO()
			gtCluster, err := manager.GetCluster(ctx, options.ClusterName, options.Namespace)
			if err != nil {
				return err
			}

			var oldReplicas int32
			if options.ComponentType == string(Frontend) {
				oldReplicas = gtCluster.Spec.Frontend.Replicas
				gtCluster.Spec.Frontend.Replicas = options.Replicas
			}

			if options.ComponentType == string(Datanode) {
				oldReplicas = gtCluster.Spec.Datanode.Replicas
				gtCluster.Spec.Datanode.Replicas = options.Replicas
			}

			log.Printf("Scaling cluster %s in %s... from %d to %d\n", options.ClusterName, options.Namespace, oldReplicas, options.Replicas)

			if err = manager.UpdateCluster(ctx, options.ClusterName, options.Namespace, gtCluster); err != nil {
				return err
			}

			log.Printf("Scaling cluster %s in %s is OK!\n", options.ClusterName, options.Namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ClusterName, "cluster-name", "n", "greptimedb", "Name of GreptimeDB cluster.")
	cmd.Flags().StringVarP(&options.ComponentType, "component-type", "c", "", "Component of GreptimeDB cluster, can be 'frontend' and 'datanode'.")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")

	return cmd
}
