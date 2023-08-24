// Copyright 2023 Greptime Team
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

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
	"github.com/GreptimeTeam/gtctl/pkg/kube"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type deployer struct {
	helmManager *helm.Manager
	client      *kube.Client
	timeout     time.Duration
	logger      logger.Logger
	dryRun      bool
}

var _ Interface = &deployer{}

type Option func(*deployer)

func NewDeployer(l logger.Logger, opts ...Option) (Interface, error) {
	hm, err := helm.NewManager(l)
	if err != nil {
		return nil, err
	}

	d := &deployer{
		helmManager: hm,
		logger:      l,
	}

	for _, opt := range opts {
		opt(d)
	}

	var client *kube.Client
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

	manifests, err := d.helmManager.LoadAndRenderChart(ctx, resourceName, resourceNamespace, helm.GreptimeDBChartName, options.GreptimeDBChartVersion, *options)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(ctx, manifests); err != nil {
		return err
	}

	return d.client.WaitForClusterReady(ctx, resourceName, resourceNamespace, d.timeout)
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

	return d.client.WaitForClusterReady(ctx, resourceName, resourceNamespace, d.timeout)
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

	// TODO(zyy17): Maybe we can set this in the top level configures.
	const (
		disableRBACConfig = "auth.rbac.create=false,auth.rbac.token.enabled=false"
	)
	options.ConfigValues += disableRBACConfig

	manifests, err := d.helmManager.LoadAndRenderChart(ctx, resourceName, resourceNamespace, helm.EtcdBitnamiOCIRegistry, helm.DefaultEtcdChartVersion, *options)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(ctx, manifests); err != nil {
		return err
	}

	return d.client.WaitForEtcdReady(ctx, resourceName, resourceNamespace, d.timeout)
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

	manifests, err := d.helmManager.LoadAndRenderChart(ctx, resourceName, resourceNamespace, helm.GreptimeDBOperatorChartName, options.GreptimeDBOperatorChartVersion, *options)
	if err != nil {
		return err
	}

	if d.dryRun {
		d.logger.V(0).Info(string(manifests))
		return nil
	}

	if err := d.client.Apply(ctx, manifests); err != nil {
		return err
	}

	return d.client.WaitForDeploymentReady(ctx, resourceName, resourceNamespace, d.timeout)
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
