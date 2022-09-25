package create

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

type createOptions struct {
	OperatorImage     string
	ClusterName       string
	Namespace         string
	DryRun            bool
	OperatorNamespace string
}

func NewCreateClusterCommand() *cobra.Command {
	var options createOptions

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create a GreptimeDB cluster.",
		Long:  `Create a GreptimeDB cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.DryRun {
				log.Printf("In dry run mode\n")
			}

			log.Printf("Creating cluster %s in %s...\n", options.ClusterName, options.Namespace)

			clusterManager, err := cluster.NewClusterManager()
			if err != nil {
				return err
			}

			operatorArgs := &cluster.OperatorDeploymentArgs{
				OperatorImage: options.OperatorImage,
				Namespace:     options.OperatorNamespace,
			}

			log.Printf("Deploying GreptimeDB Operator ...\n")
			if err := clusterManager.DeployOperator(operatorArgs, options.DryRun); err != nil {
				return err
			}
			log.Printf("GreptimeDB Operator is Ready!\n")

			log.Printf("Deploying GreptimeDB Cluster ...\n")
			dbArgs := &cluster.DBDeploymentArgs{
				CluserName: options.ClusterName,
				Namespace:  options.Namespace,
			}
			if err := clusterManager.DeployCluster(dbArgs, options.DryRun); err != nil {
				return err
			}
			log.Printf("GreptimeDB Cluster is Ready!\n")
			log.Printf("You can use `kubectl port-forward svc/%s-frontend -n %s 3306:3306` to access the database.\n", options.ClusterName, options.Namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.OperatorImage, "operator-image", "", "Image of GreptimeDB operator.")
	cmd.Flags().StringVar(&options.OperatorNamespace, "operator-namespace", "default", "The namespace of deploying greptimedb-operator.")
	cmd.Flags().StringVarP(&options.ClusterName, "cluster-name", "n", "greptimedb", "Name of GreptimeDB cluster.")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Output the manifests without applying them.")

	return cmd
}
