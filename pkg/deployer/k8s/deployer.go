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

package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/strvals"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type deployer struct {
	render  *helm.Render
	client  *kube.Client
	timeout time.Duration
	logger  logger.Logger
	dryRun  bool
}

var _ Deployer = &deployer{}

type Option func(*deployer)

func NewDeployer(l logger.Logger, opts ...Option) (Deployer, error) {
	d := &deployer{
		render: &helm.Render{},
		logger: l,
	}

	for _, opt := range opts {
		opt(d)
	}

	var (
		client *kube.Client
		err    error
	)
	if !d.dryRun {
		client, err = kube.NewClient("")
		if err != nil {
			return nil, err
		}
	}

	d.client = client

	return d, nil
}

func WithDryRun(dryRun bool) Option {
	return func(d *deployer) {
		d.dryRun = dryRun
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(d *deployer) {
		d.timeout = timeout
	}
}

func (d *deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return nil, err
	}

	cluster, err := d.client.GetCluster(ctx, resourceName, resourceNamespace)
	if err != nil {
		return nil, err
	}

	return &GreptimeDBCluster{
		Raw: cluster,
	}, nil
}

func (d *deployer) ListGreptimeDBClusters(ctx context.Context, options *ListGreptimeDBClustersOptions) ([]*GreptimeDBCluster, error) {
	clusters, err := d.client.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var result []*GreptimeDBCluster
	for _, cluster := range clusters.Items {
		result = append(result, &GreptimeDBCluster{
			Raw: &cluster,
		})
	}

	return result, nil
}

func (d *deployer) CreateGreptimeDBCluster(ctx context.Context, name string, options *CreateGreptimeDBClusterOptions) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}

	values, err := d.generateClusterValues(options)
	if err != nil {
		return err
	}

	downloadURL, err := d.getChartDownloadURL(GreptimeDBChartName, options.GreptimeDBChartVersion)
	if err != nil {
		return err
	}

	chart, err := d.render.LoadChartFromRemoteCharts(downloadURL)
	if err != nil {
		return err
	}

	manifests, err := d.render.GenerateManifests(resourceName, resourceNamespace, chart, values)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(manifests); err != nil {
		return err
	}

	return d.client.WaitForClusterReady(resourceName, resourceNamespace, d.timeout)
}

func (d *deployer) UpdateGreptimeDBCluster(ctx context.Context, name string, options *UpdateGreptimeDBClusterOptions) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}

	newCluster, ok := options.NewCluster.Raw.(*greptimedbclusterv1alpha1.GreptimeDBCluster)
	if !ok {
		return fmt.Errorf("invalid cluster type")
	}

	if err := d.client.UpdateCluster(ctx, resourceNamespace, newCluster); err != nil {
		return err
	}

	return d.client.WaitForClusterReady(resourceName, resourceNamespace, d.timeout)
}

func (d *deployer) DeleteGreptimeDBCluster(ctx context.Context, name string, options *DeleteGreptimeDBClusterOption) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}
	return d.client.DeleteCluster(ctx, resourceName, resourceNamespace)
}

func (d *deployer) CreateEtcdCluster(ctx context.Context, name string, options *CreateEtcdClusterOptions) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}

	values, err := d.generateEtcdValues(options)
	if err != nil {
		return err
	}

	downloadURL, err := d.getChartDownloadURL(GreptimeDBEtcdChartName, options.EtcdChartVersion)
	if err != nil {
		return err
	}

	chart, err := d.render.LoadChartFromRemoteCharts(downloadURL)
	if err != nil {
		return err
	}

	manifests, err := d.render.GenerateManifests(resourceName, resourceNamespace, chart, values)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(manifests); err != nil {
		return err
	}

	return d.client.WaitForEtcdReady(resourceName, resourceNamespace, d.timeout)
}

func (d *deployer) DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}

	return d.client.DeleteEtcdCluster(ctx, resourceName, resourceNamespace)
}

func (d *deployer) CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error {
	resourceNamespace, resourceName, err := d.splitNamescapedName(name)
	if err != nil {
		return err
	}

	values, err := d.generateOperatorValues(options)
	if err != nil {
		return err
	}

	downloadURL, err := d.getChartDownloadURL(GreptimeDBOperatorChartName, options.GreptimeDBOperatorChartVersion)
	if err != nil {
		return err
	}

	chart, err := d.render.LoadChartFromRemoteCharts(downloadURL)
	if err != nil {
		return err
	}

	manifests, err := d.render.GenerateManifests(GreptimeDBOperatorChartName, resourceNamespace, chart, values)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(manifests); err != nil {
		return err
	}

	return d.client.WaitForDeploymentReady(resourceName, resourceNamespace, d.timeout)
}

func (d *deployer) splitNamescapedName(name string) (string, string, error) {
	if name == "" {
		return "", "", fmt.Errorf("empty namespaced name")
	}

	split := strings.Split(name, "/")
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid namespaced name '%s'", name)
	}

	return split[0], split[1], nil
}

func (d *deployer) generateClusterValues(options *CreateGreptimeDBClusterOptions) (map[string]interface{}, error) {
	var rawArgs []string

	// TODO(zyy17): It's very ugly to generate Helm values...
	if len(options.ImageRegistry) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("image.registry=%s", options.ImageRegistry))
	}

	if len(options.DatanodeStorageClassName) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageClassName=%s", options.DatanodeStorageClassName))
	}

	if len(options.DatanodeStorageSize) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageSize=%s", options.DatanodeStorageSize))
	}

	if len(options.DatanodeStorageRetainPolicy) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("datanode.storage.storageRetainPolicy=%s", options.DatanodeStorageRetainPolicy))
	}

	if len(options.EtcdEndPoint) > 0 {
		rawArgs = append(rawArgs, fmt.Sprintf("etcdEndpoints=%s", options.EtcdEndPoint))
	}

	if len(rawArgs) > 0 {
		values, err := d.generateHelmValues(strings.Join(rawArgs, ","))
		if err != nil {
			return nil, err
		}

		d.logger.V(3).Infof("generate greptimedb cluster helm values: '%v'", values)

		return values, nil
	}

	return nil, nil
}

func (d *deployer) generateOperatorValues(options *CreateGreptimeDBOperatorOptions) (map[string]interface{}, error) {
	// TODO(zyy17): It's very ugly to generate Helm values...
	if len(options.ImageRegistry) > 0 {
		values, err := d.generateHelmValues(fmt.Sprintf("image.registry=%s", options.ImageRegistry))
		if err != nil {
			return nil, err
		}

		d.logger.V(3).Infof("generate greptimedb operator helm values: '%v'", values)
		return values, nil
	}

	return nil, nil
}

func (d *deployer) generateEtcdValues(options *CreateEtcdClusterOptions) (map[string]interface{}, error) {
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
		values, err := d.generateHelmValues(strings.Join(rawArgs, ","))
		if err != nil {
			return nil, err
		}

		d.logger.V(3).Infof("generate etcd helm values: '%v'", values)
		return values, nil
	}

	return nil, nil
}

func (d *deployer) generateHelmValues(args string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if err := strvals.ParseInto(args, values); err != nil {
		return nil, err
	}
	return values, nil
}

func (d *deployer) getChartDownloadURL(chartName, version string) (string, error) {
	indexFile, err := d.render.GetIndexFile(GreptimeChartIndexURL)
	if err != nil {
		return "", err
	}

	var downloadURL string
	if version == "" {
		chartVersion, err := d.render.GetLatestChart(indexFile, chartName)
		if err != nil {
			return "", err
		}
		downloadURL = chartVersion.URLs[0]
		d.logger.V(3).Infof("get latest chart '%s', version '%s', url: '%s'",
			chartName, chartVersion.Version, downloadURL)
	} else {
		// The download URL example: 'https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-0.1.1-alpha.3/greptimedb-0.1.1-alpha.3.tgz'.
		chartName := chartName + "-" + version
		downloadURL = fmt.Sprintf("%s/%s/%s.tgz", GreptimeChartReleaseDownloadURL, chartName, chartName)
		d.logger.V(3).Infof("get given version chart '%s', version '%s', url: '%s'",
			chartName, version, downloadURL)
	}

	return downloadURL, nil
}
