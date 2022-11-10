package manager

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/strvals"
	"strings"
	"time"

	greptimedbv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	"github.com/GreptimeTeam/gtctl/pkg/log"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

// Manager manage the cluster resources.
type Manager interface {
	GetCluster(ctx context.Context, options *GetClusterOptions) (*greptimedbv1alpha1.GreptimeDBCluster, error)
	ListClusters(ctx context.Context, options *ListClusterOptions) (*greptimedbv1alpha1.GreptimeDBClusterList, error)
	CreateCluster(ctx context.Context, options *CreateClusterOptions) error
	UpdateCluster(ctx context.Context, options *UpdateClusterOptions) error
	DeleteCluster(ctx context.Context, options *DeleteClusterOption) error
	CreateOperator(ctx context.Context, options *CreateOperatorOptions) error
}

type GetClusterOptions struct {
	ClusterName string
	Namespace   string
}

type ListClusterOptions struct{}

type CreateClusterOptions struct {
	ClusterName   string
	Namespace     string
	MetaImage     string
	FrontendImage string
	DatanodeImage string
	EtcdImage     string
	Timeout       time.Duration
	DryRun        bool
}

type UpdateClusterOptions struct {
	ClusterName string
	Namespace   string
	Timeout     time.Duration
	NewCluster  *greptimedbv1alpha1.GreptimeDBCluster
}

type DeleteClusterOption struct {
	ClusterName  string
	Namespace    string
	TearDownEtcd bool
}

type CreateOperatorOptions struct {
	OperatorImage string
	Namespace     string

	Timeout time.Duration

	DryRun bool
}

var _ Manager = &manager{}

func New(l log.Logger, dryRun bool) (Manager, error) {
	var (
		client *kube.Client
		err    error
	)

	if !dryRun {
		client, err = kube.NewClient("")
		if err != nil {
			return nil, err
		}
	}

	return &manager{
		render: &helm.Render{},
		client: client,
		l:      l,
	}, nil
}

type manager struct {
	render *helm.Render
	client *kube.Client
	l      log.Logger
}

func (m *manager) GetCluster(ctx context.Context, options *GetClusterOptions) (*greptimedbv1alpha1.GreptimeDBCluster, error) {
	return m.client.GetCluster(ctx, options.ClusterName, options.Namespace)
}

func (m *manager) ListClusters(ctx context.Context, options *ListClusterOptions) (*greptimedbv1alpha1.GreptimeDBClusterList, error) {
	return m.client.ListCluster(ctx)
}

func (m *manager) CreateCluster(ctx context.Context, options *CreateClusterOptions) error {
	values, err := m.generateClusterValues(options)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromEmbedCharts(defaultGreptimeDBHelmPackageName, defaultGreptimeDBHelmPackageVersion)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(options.ClusterName, options.Namespace, chart, values)
	if err != nil {
		return err
	}

	if options.DryRun {
		m.l.Info(string(manifests))
		return nil
	}

	if err := m.client.Apply(manifests); err != nil {
		return err
	}

	if err := m.client.WaitForClusterReady(options.ClusterName, options.Namespace, options.Timeout); err != nil {
		return err
	}

	return nil
}

func (m *manager) UpdateCluster(ctx context.Context, options *UpdateClusterOptions) error {
	if err := m.client.UpdateCluster(ctx, options.Namespace, options.NewCluster); err != nil {
		return err
	}

	if err := m.client.WaitForClusterReady(options.ClusterName, options.Namespace, options.Timeout); err != nil {
		return err
	}

	return nil
}

func (m *manager) DeleteCluster(ctx context.Context, options *DeleteClusterOption) error {
	if err := m.client.DeleteCluster(ctx, options.ClusterName, options.Namespace); err != nil {
		return err
	}

	if options.TearDownEtcd {
		if err := m.client.DeleteEtcdCluster(ctx, options.ClusterName, options.Namespace); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) CreateOperator(ctx context.Context, options *CreateOperatorOptions) error {
	values, err := m.generateOperatorValues(options)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromEmbedCharts(defaultOperatorHelmPackageName, defaultOperatorHelmPackageVersion)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(defaultOperatorReleaseName, options.Namespace, chart, values)
	if err != nil {
		return err
	}

	if options.DryRun {
		m.l.Infof(string(manifests))
		return nil
	}

	if err := m.client.Apply(manifests); err != nil {
		return err
	}

	if err := m.client.WaitForDeploymentReady(defaultOperatorReleaseName, options.Namespace, options.Timeout); err != nil {
		return err
	}

	return nil
}

func (m *manager) generateClusterValues(options *CreateClusterOptions) (map[string]interface{}, error) {
	var rawArgs []string

	// TODO(zyy17): It's very ugly to generate Helm values...
	if len(options.FrontendImage) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("frontend.main.image=%s", options.FrontendImage))
	}

	if len(options.MetaImage) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("meta.main.image=%s", options.MetaImage))
	}

	if len(options.DatanodeImage) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.main.image=%s", options.DatanodeImage))
	}

	if len(options.EtcdImage) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("etcd.image=%s", options.EtcdImage))
	}

	if len(rawArgs) > 0 {
		values, err := m.generateHelmValues(strings.Join(rawArgs, ","))
		if err != nil {
			return nil, err
		}
		return values, nil
	}

	return nil, nil
}

func (m *manager) generateOperatorValues(options *CreateOperatorOptions) (map[string]interface{}, error) {
	// TODO(zyy17): It's very ugly to generate Helm values...
	if len(options.OperatorImage) > 0 {
		repo, tag := utils.SplitImageURL(options.OperatorImage)
		values, err := m.generateHelmValues(fmt.Sprintf("image.repository=%s,image.tag=%s", repo, tag))
		if err != nil {
			return nil, err
		}
		return values, nil
	}

	return nil, nil
}

func (m *manager) generateHelmValues(args string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if err := strvals.ParseInto(args, values); err != nil {
		return nil, err
	}
	return values, nil
}
