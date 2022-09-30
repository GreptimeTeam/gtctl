package create

import (
	"log"
	"time"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/cluster"
)

type createOptions struct {
	OperatorImage     string
	ClusterName       string
	Namespace         string
	OperatorNamespace string
	MetaImage         string
	FrontendImage     string
	DatanodeImage     string
	EtcdImage         string

	DryRun  bool
	Timeout int
}

const (
	defaultGreptimeDBOperatorImage = "grygt/operator:latest"
	defaultMetaImage               = "grygt/meta:latest"
	defaultFrontendImage           = "grygt/db:latest"
	defaultDatanodeImage           = "grygt/db:latest"
	defaultEtcdImage               = "grygt/etcd:latest"
)

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
				Timeout:       time.Duration(options.Timeout) * time.Second,
			}

			log.Printf("Deploying GreptimeDB Operator ...\n")
			if err := clusterManager.DeployOperator(operatorArgs, options.DryRun); err != nil {
				return err
			}
			log.Printf("GreptimeDB Operator is Ready!\n")

			log.Printf("Deploying GreptimeDB Cluster ...\n")
			dbArgs := &cluster.DBDeploymentArgs{
				CluserName:    options.ClusterName,
				Namespace:     options.Namespace,
				FrontendImage: options.FrontendImage,
				MetaImage:     options.MetaImage,
				DatanodeImage: options.DatanodeImage,
				EtcdImage:     options.EtcdImage,
				Timeout:       time.Duration(options.Timeout) * time.Second,
			}
			if err := clusterManager.DeployCluster(dbArgs, options.DryRun); err != nil {
				return err
			}
			log.Printf("GreptimeDB Cluster is Ready!\n")
			log.Printf("You can use `kubectl port-forward svc/%s-frontend -n %s 3306:3306` to access the database.\n", options.ClusterName, options.Namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.OperatorImage, "operator-image", defaultGreptimeDBOperatorImage, "Image of GreptimeDB operator.")
	cmd.Flags().StringVar(&options.OperatorNamespace, "operator-namespace", "default", "The namespace of deploying greptimedb-operator.")
	cmd.Flags().StringVar(&options.FrontendImage, "frontend-image", defaultFrontendImage, "Image of Frontend component.")
	cmd.Flags().StringVar(&options.MetaImage, "meta-image", defaultMetaImage, "Image of Meta component.")
	cmd.Flags().StringVar(&options.DatanodeImage, "datanode-image", defaultDatanodeImage, "Image of Datanode component.")
	cmd.Flags().StringVar(&options.EtcdImage, "etcd-image", defaultEtcdImage, "Image of etcd.")
	cmd.Flags().StringVar(&options.ClusterName, "name", "greptimedb", "Name of GreptimeDB cluster.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Output the manifests without applying them.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")

	return cmd
}
