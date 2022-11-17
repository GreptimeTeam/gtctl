package create

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/manager"
)

type createClusterCliOptions struct {
	Namespace           string
	OperatorNamespace   string
	StorageClassName    string
	StorageSize         string
	StorageRetainPolicy string
	GreptimeDBVersion   string
	OperatorVersion     string
	ImageRegistry       string
	RepoName            string

	DryRun  bool
	Timeout int
}

func NewCreateClusterCommand(l log.Logger) *cobra.Command {
	var options createClusterCliOptions

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a GreptimeDB cluster",
		Long:  `Create a GreptimeDB cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name should be set")
			}

			m, err := manager.New(l, options.DryRun)
			if err != nil {
				return err
			}

			createOperatorOptions := &manager.CreateOperatorOptions{
				Namespace:       options.OperatorNamespace,
				Timeout:         time.Duration(options.Timeout) * time.Second,
				DryRun:          options.DryRun,
				OperatorVersion: options.OperatorVersion,
				ImageRegistry:   options.ImageRegistry,
				RepoName:        options.RepoName,
			}

			var (
				clusterName = args[0]

				// TODO(zyy17): should use timeout context.
				ctx = context.TODO()
			)

			l.Infof("☕️ Creating cluster '%s' in namespace '%s'...\n", log.Bold(clusterName), log.Bold(options.Namespace))
			l.Infof("☕️ Start to create greptimedb-operator...\n")
			if err := log.StartSpinning("Creating greptimedb-operator...", func() error {
				return m.CreateOperator(ctx, createOperatorOptions)
			}); err != nil {
				return err
			}
			l.Infof("🎉 Finish to create greptimedb-operator.\n")

			l.Infof("☕️ Start to create GreptimeDB cluster...\n")
			createClusterOptions := &manager.CreateClusterOptions{
				ClusterName:         args[0],
				Namespace:           options.Namespace,
				StorageClassName:    options.StorageClassName,
				StorageSize:         options.StorageSize,
				StorageRetainPolicy: options.StorageRetainPolicy,
				Timeout:             time.Duration(options.Timeout) * time.Second,
				DryRun:              options.DryRun,
				GreptimeDBVersion:   options.GreptimeDBVersion,
				ImageRegistry:       options.ImageRegistry,
				RepoName:            options.RepoName,
			}

			if err := log.StartSpinning("Creating GreptimeDB cluster", func() error {
				return m.CreateCluster(ctx, createClusterOptions)
			}); err != nil {
				return err
			}

			if !options.DryRun {
				l.Infof("🎉 Finish to create GreptimeDB cluster '%s'.\n", log.Bold(clusterName))
				l.Infof("💡 You can use `%s` to access the database.\n", log.Bold(fmt.Sprintf("kubectl port-forward svc/%s-frontend -n %s 4002:4002", clusterName, options.Namespace)))
				l.Infof("😊 Thank you for using %s!\n", log.Bold("GreptimeDB"))
				l.Infof("🔑 %s\n", log.Bold("Invest in Data, Harvest over Time."))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&options.OperatorNamespace, "operator-namespace", "default", "The namespace of deploying greptimedb-operator.")
	cmd.Flags().StringVar(&options.StorageClassName, "storage-class-name", "standard", "Datanode storage class name.")
	cmd.Flags().StringVar(&options.StorageSize, "storage-size", "10Gi", "Datanode persistent volume size.")
	cmd.Flags().StringVar(&options.StorageRetainPolicy, "retain-policy", "Retain", "Datanode pvc retain policy.")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace of GreptimeDB cluster.")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Output the manifests without applying them.")
	cmd.Flags().IntVar(&options.Timeout, "timeout", -1, "Timeout in seconds for the command to complete, default is no timeout.")
	cmd.Flags().StringVar(&options.GreptimeDBVersion, "version", manager.DefaultGreptimeDBChartVersion, "The GreptimeDB version.")
	cmd.Flags().StringVar(&options.OperatorVersion, "operator-version", manager.DefaultGreptimeDBOperatorChartVersion, "The greptimedb-operator version.")
	cmd.Flags().StringVar(&options.ImageRegistry, "image-registry", "", "The image registry")
	cmd.Flags().StringVar(&options.RepoName, "repo-name", "greptime", "The image repo name")

	return cmd
}
