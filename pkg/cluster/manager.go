package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/strvals"

	greptimedbv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
)

const (
	defaultOperatorReleaseName = "greptimedb-operator"

	defaultGreptimeDBOperatorHelmPackageName = "greptimedb-operator"
	defaultGreptimeDBHelmPackageName         = "greptimedb"

	defaultGreptimeDBOperatorHelmPackageVersion = "0.1.0"
	defaultGreptimeDBHelmPackageVersion         = "0.1.0"
)

type Manager struct {
	render *helm.Render
	client *kube.Client
}

type OperatorDeploymentArgs struct {
	OperatorImage string
	Namespace     string

	Timeout time.Duration
}

type DBDeploymentArgs struct {
	ClusterName   string
	Namespace     string
	MetaImage     string
	FrontendImage string
	DatanodeImage string
	EtcdImage     string

	Timeout time.Duration
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

func (m *Manager) GetAllClusters(ctx context.Context) (*greptimedbv1alpha1.GreptimeDBClusterList, error) {
	return m.client.GetAllClusters(ctx)
}

func (m *Manager) UpdateCluster(ctx context.Context, name, namespace string, newCluster *greptimedbv1alpha1.GreptimeDBCluster, timeout time.Duration) error {
	if err := m.client.UpdateCluster(ctx, namespace, newCluster); err != nil {
		return err
	}

	if err := m.client.WaitForClusterReady(name, namespace, timeout); err != nil {
		return err
	}

	return nil
}

func (m *Manager) DeployOperator(args *OperatorDeploymentArgs, dryRun bool) error {
	repo, tag := splitImageURL(args.OperatorImage)
	values, err := m.generateHelmValues(fmt.Sprintf("image.repository=%s,image.tag=%s", repo, tag))
	if err != nil {
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

	if err := m.client.WaitForDeploymentReady(defaultOperatorReleaseName, args.Namespace, args.Timeout); err != nil {
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
	// TODO(zyy17): It's very ugly to generate Helm values...
	rawArgs := fmt.Sprintf("frontend.main.image=%s,meta.main.image=%s,datanode.main.image=%s,etcd.image=%s",
		args.FrontendImage, args.MetaImage, args.DatanodeImage, args.EtcdImage)
	values, err := m.generateHelmValues(rawArgs)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromEmbedCharts(defaultGreptimeDBHelmPackageName, defaultGreptimeDBHelmPackageVersion)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(args.ClusterName, args.Namespace, chart, values)
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

	if err := m.client.WaitForClusterReady(args.ClusterName, args.Namespace, args.Timeout); err != nil {
		return err
	}

	return nil
}

func (m *Manager) generateHelmValues(args string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if err := strvals.ParseInto(args, values); err != nil {
		return nil, err
	}
	return values, nil
}

// TODO(zyy17): validation?
func splitImageURL(imageURL string) (string, string) {
	split := strings.Split(imageURL, ":")
	if len(split) != 2 {
		return "", ""
	}

	return split[0], split[1]
}
