package scale

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/log"
)

type scaleOptions struct {
	Namespace     string
	ComponentType string
	Replicas      int32
	Timeout       int
}

// FIXME(zyy17): ComponentType should be defined in CRDs.

type ComponentType string

const (
	Frontend ComponentType = "frontend"
	Datanode ComponentType = "datanode"
)

func NewScaleClusterCommand(l log.Logger) *cobra.Command {
	var options scaleOptions

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Scale GreptimeDB cluster.",
		Long:  `Scale GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}
			if options.ComponentType == "" {
				return fmt.Errorf("component type is required")
			}

			if options.ComponentType != string(Frontend) && options.ComponentType != string(Datanode) {
				return fmt.Errorf("component type is invalid")
			}

			if options.Replicas < 1 {
				return fmt.Errorf("replicas should be equal or greater than 1")
			}

			manager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			ctx := context.TODO()
			gtCluster, err := manager.GetCluster(ctx, args[0], options.Namespace)
			if err != nil && errors.IsNotFound(err) {
				l.Infof("cluster %s in %s not found\n", args[0], options.Namespace)
				return nil
			} else if err != nil {
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

			l.Infof("Scaling cluster %s in %s... from %d to %d\n", args[0], options.Namespace, oldReplicas, options.Replicas)

			if err = manager.UpdateCluster(ctx, args[0], options.Namespace, gtCluster, time.Duration(options.Timeout)*time.Second); err != nil {
				return err
			}

			l.Infof("Scaling cluster %s in %s is OK!\n", args[0], options.Namespace)

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.ComponentType, "component-type", "c", "", "Component of GreptimeDB cluster, can be 'frontend' and 'datanode'.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().Int32Var(&options.Replicas, "replicas", 0, "The replicas of component of GreptimeDB cluster.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}
