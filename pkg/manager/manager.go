// Copyright 2022 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/strvals"

	greptimedbv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	"github.com/GreptimeTeam/gtctl/pkg/log"
)

const (
	GreptimeChartIndexURL           = "https://raw.githubusercontent.com/GreptimeTeam/helm-charts/gh-pages/index.yaml"
	GreptimeChartReleaseDownloadURL = "https://github.com/GreptimeTeam/helm-charts/releases/download"
)

// Manager manage the cluster resources.
type Manager interface {
	GetCluster(ctx context.Context, options *GetClusterOptions) (*greptimedbv1alpha1.GreptimeDBCluster, error)
	ListClusters(ctx context.Context, options *ListClusterOptions) (*greptimedbv1alpha1.GreptimeDBClusterList, error)
	CreateCluster(ctx context.Context, options *CreateClusterOptions) error
	UpdateCluster(ctx context.Context, options *UpdateClusterOptions) error
	DeleteCluster(ctx context.Context, options *DeleteClusterOption) error
	CreateOperator(ctx context.Context, options *CreateOperatorOptions) error
	DeleteEtcdCluster(ctx context.Context, options *DeleteEtcdClusterOption) error
	CreateEtcdCluster(ctx context.Context, options *CreateEtcdOptions) error
}

type GetClusterOptions struct {
	ClusterName string
	Namespace   string
}

type ListClusterOptions struct{}

type CreateClusterOptions struct {
	ClusterName            string
	Namespace              string
	StorageClassName       string
	StorageSize            string
	StorageRetainPolicy    string
	GreptimeDBChartVersion string
	ImageRegistry          string
	EtcdEndPoint           string

	Timeout time.Duration
	DryRun  bool
}

type CreateEtcdOptions struct {
	Name                 string
	Namespace            string
	ImageRegistry        string
	EtcdChartVersion     string
	EtcdStorageClassName string
	EtcdStorageSize      string

	Timeout time.Duration
	DryRun  bool
}

type UpdateClusterOptions struct {
	ClusterName string
	Namespace   string
	Timeout     time.Duration
	NewCluster  *greptimedbv1alpha1.GreptimeDBCluster
}

type DeleteClusterOption struct {
	ClusterName string
	Namespace   string
}

type DeleteEtcdClusterOption struct {
	Name      string
	Namespace string
}

type CreateOperatorOptions struct {
	Namespace              string
	GreptimeDBChartVersion string
	ImageRegistry          string

	Timeout time.Duration
	DryRun  bool
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

	downloadURL, err := m.getChartDownloadURL(defaultGreptimeDBHelmPackageName, options.GreptimeDBChartVersion)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromRemoteCharts(downloadURL)
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

	return m.client.WaitForClusterReady(options.ClusterName, options.Namespace, options.Timeout)
}

func (m *manager) UpdateCluster(ctx context.Context, options *UpdateClusterOptions) error {
	if err := m.client.UpdateCluster(ctx, options.Namespace, options.NewCluster); err != nil {
		return err
	}

	return m.client.WaitForClusterReady(options.ClusterName, options.Namespace, options.Timeout)
}

func (m *manager) DeleteCluster(ctx context.Context, options *DeleteClusterOption) error {
	return m.client.DeleteCluster(ctx, options.ClusterName, options.Namespace)
}

func (m *manager) DeleteEtcdCluster(ctx context.Context, options *DeleteEtcdClusterOption) error {
	return m.client.DeleteEtcdCluster(ctx, options.Name, options.Namespace)
}

func (m *manager) CreateOperator(ctx context.Context, options *CreateOperatorOptions) error {
	values, err := m.generateOperatorValues(options)
	if err != nil {
		return err
	}

	downloadURL, err := m.getChartDownloadURL(defaultOperatorHelmPackageName, options.GreptimeDBChartVersion)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromRemoteCharts(downloadURL)
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

	return m.client.WaitForDeploymentReady(defaultOperatorReleaseName, options.Namespace, options.Timeout)
}

func (m *manager) CreateEtcdCluster(ctx context.Context, options *CreateEtcdOptions) error {
	values, err := m.generateEtcdValues(options)
	if err != nil {
		return err
	}

	downloadURL, err := m.getChartDownloadURL(defaultEtcdHelmPackageName, options.EtcdChartVersion)
	if err != nil {
		return err
	}

	chart, err := m.render.LoadChartFromRemoteCharts(downloadURL)
	if err != nil {
		return err
	}

	manifests, err := m.render.GenerateManifests(options.Name, options.Namespace, chart, values)
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

	return m.client.WaitForEtcdReady(options.Name, options.Namespace, options.Timeout)
}

func (m *manager) generateClusterValues(options *CreateClusterOptions) (map[string]interface{}, error) {
	var rawArgs []string

	// TODO(zyy17): It's very ugly to generate Helm values...
	if len(options.ImageRegistry) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("image.registry=%s", options.ImageRegistry))
	}

	if len(options.StorageClassName) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageClassName=%s", options.StorageClassName))
	}

	if len(options.StorageSize) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageSize=%s", options.StorageSize))
	}

	if len(options.StorageRetainPolicy) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageRetainPolicy=%s", options.StorageRetainPolicy))
	}

	if len(options.EtcdEndPoint) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("etcdEndpoints=%s", options.EtcdEndPoint))
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
	if len(options.ImageRegistry) > 0 {
		values, err := m.generateHelmValues(fmt.Sprintf("image.registry=%s", options.ImageRegistry))
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

func (m *manager) generateEtcdValues(options *CreateEtcdOptions) (map[string]interface{}, error) {
	var rawArgs []string
	if len(options.ImageRegistry) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("image.registry=%s", options.ImageRegistry))
	}

	if len(options.EtcdStorageClassName) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("storage.storageClassName=%s", options.EtcdStorageClassName))
	}

	if len(options.EtcdStorageSize) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("storage.volumeSize=%s", options.EtcdStorageSize))
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

func (m *manager) getChartDownloadURL(chartName, version string) (string, error) {
	indexFile, err := m.render.GetIndexFile(GreptimeChartIndexURL)
	if err != nil {
		return "", err
	}

	var downloadURL string
	if version == "" {
		chartVersion, err := m.render.GetLatestChart(indexFile, chartName)
		if err != nil {
			return "", err
		}
		downloadURL = chartVersion.URLs[0]
	} else {
		// The download URL example: 'https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-0.1.1-alpha.3/greptimedb-0.1.1-alpha.3.tgz'.
		chartName := chartName + "-" + version
		downloadURL = fmt.Sprintf("%s/%s/%s.tgz", GreptimeChartReleaseDownloadURL, chartName, chartName)
	}

	return downloadURL, nil
}
