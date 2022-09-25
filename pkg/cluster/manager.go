package cluster

import (
	"context"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/strvals"

	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	greptimedbv1alpha1 "github.com/GreptimeTeam/gtctl/third_party/apis/v1alpha1"
)

const (
	defaultOperatorReleaseName = "greptimedb-operator"

	defaultGreptimeDBOperatorHelmPackageName = "greptimedb-operator"
	defaultGreptimeDBHelmPackageName         = "greptimedb"

	defaultGreptimeDBOperatorHelmPackageVersion = "0.1.0"
	defaultGreptimeDBHelmPackageVersion         = "0.1.0"

	defaultPollingTimeout = 120 * time.Second
)

type Manager struct {
	render *helm.Render
	client *kube.Client
}

type OperatorDeploymentArgs struct {
	OperatorImage string
	Namespace     string
}

type DBDeploymentArgs struct {
	CluserName string
	Namespace  string
}

func NewClusterManager() (*Manager, error) {
	client, err := kube.NewClient("")
	if err != nil {
		return nil, err
	}

	return &Manager{
		render: &helm.Render{},
		client: client,
	}, nil
}

func (m *Manager) GetCluster(ctx context.Context, name, namespace string) (*greptimedbv1alpha1.GreptimeDBCluster, error) {
	return m.client.GetCluster(ctx, name, namespace)
}

func (m *Manager) UpdateCluster(ctx context.Context, name, namespace string, newCluster *greptimedbv1alpha1.GreptimeDBCluster) error {
	if err := m.client.UpdateCluster(ctx, namespace, newCluster); err != nil {
		return err
	}

	if err := m.client.WaitForClusterReady(name, namespace, defaultPollingTimeout); err != nil {
		return err
	}

	return nil
}

func (m *Manager) DeployOperator(args *OperatorDeploymentArgs, dryRun bool) error {
	var formatedArgs string
	if len(args.OperatorImage) > 0 {
		formatedArgs = fmt.Sprintf("greptimedbOperator.image=%s", args.OperatorImage)
	}
	values := make(map[string]interface{})
	if err := strvals.ParseInto(formatedArgs, values); err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromEmbedCharts(defaultGreptimeDBOperatorHelmPackageName, defaultGreptimeDBOperatorHelmPackageVersion)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(defaultOperatorReleaseName, args.Namespace, chart, values)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(string(manifests))
		return nil
	}

	if err := m.client.Apply(manifests); err != nil {
		return err
	}

	if err := m.client.WaitForDeploymentReady(defaultOperatorReleaseName, args.Namespace, defaultPollingTimeout); err != nil {
		return err
	}

	return nil
}

func (m *Manager) DeleteCluster(ctx context.Context, name, namespace string, tearDownEtcd bool) error {
	if err := m.client.DeleteCluster(ctx, name, namespace); err != nil {
		return err
	}

	if tearDownEtcd {
		if err := m.client.DeleteEtcdCluster(ctx, name, namespace); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) DeployCluster(args *DBDeploymentArgs, dryRun bool) error {
	values := make(map[string]interface{})

	chart, err := m.render.LoadChartFromEmbedCharts(defaultGreptimeDBHelmPackageName, defaultGreptimeDBHelmPackageVersion)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(args.CluserName, args.Namespace, chart, values)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(string(manifests))
		return nil
	}

	if err := m.client.Apply(manifests); err != nil {
		return err
	}

	if err := m.client.WaitForClusterReady(args.CluserName, args.Namespace, defaultPollingTimeout); err != nil {
		return err
	}

	return nil
}
