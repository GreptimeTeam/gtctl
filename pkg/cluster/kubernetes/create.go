// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"
	"fmt"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/helm"
)

const (
	AliCloudRegistry = "greptime-registry.cn-hangzhou.cr.aliyuncs.com"

	disableRBACConfig = "auth.rbac.create=false,auth.rbac.token.enabled=false,"
)

func (c *Cluster) Create(ctx context.Context, options *opt.CreateOptions) error {
	spinner := options.Spinner

	withSpinner := func(target string, f func(context.Context, *opt.CreateOptions) error) error {
		if !c.dryRun && spinner != nil {
			spinner.Start(fmt.Sprintf("Installing %s...", target))
		}

		if err := f(ctx, options); err != nil {
			if spinner != nil {
				spinner.Stop(false, fmt.Sprintf("Installing %s failed", target))
			}
			return err
		}

		if !c.dryRun {
			if spinner != nil {
				spinner.Stop(true, fmt.Sprintf("Installing %s successfully 🎉", target))
			}
		}
		return nil
	}

	if err := withSpinner("GreptimeDB Operator", c.createOperator); err != nil {
		return err
	}
	if err := withSpinner("Etcd cluster", c.createEtcdCluster); err != nil {
		return err
	}
	if err := withSpinner("GreptimeDB cluster", c.createCluster); err != nil {
		return err
	}

	return nil
}

// createOperator creates GreptimeDB Operator.
func (c *Cluster) createOperator(ctx context.Context, options *opt.CreateOptions) error {
	if options.Operator == nil {
		return fmt.Errorf("missing create greptimedb operator options")
	}
	operatorOpt := options.Operator
	resourceName, resourceNamespace := OperatorName(), options.Namespace

	if operatorOpt.UseGreptimeCNArtifacts && len(operatorOpt.ImageRegistry) == 0 {
		operatorOpt.ConfigValues += fmt.Sprintf("image.registry=%s,", AliCloudRegistry)
	}

	opts := &helm.LoadOptions{
		ReleaseName:   resourceName,
		Namespace:     resourceNamespace,
		ChartName:     artifacts.GreptimeDBOperatorChartName,
		ChartVersion:  operatorOpt.GreptimeDBOperatorChartVersion,
		FromCNRegion:  operatorOpt.UseGreptimeCNArtifacts,
		ValuesOptions: *operatorOpt,
		EnableCache:   true,
		ValuesFile:    operatorOpt.ValuesFile,
	}
	manifests, err := c.helmLoader.LoadAndRenderChart(ctx, opts)
	if err != nil {
		return err
	}

	if c.dryRun {
		c.logger.V(0).Info(string(manifests))
		return nil
	}

	if err = c.client.Apply(ctx, manifests); err != nil {
		return err
	}

	return c.client.WaitForDeploymentReady(ctx, resourceName, resourceNamespace, c.timeout)
}

// createCluster creates GreptimeDB cluster.
func (c *Cluster) createCluster(ctx context.Context, options *opt.CreateOptions) error {
	if options.Cluster == nil {
		return fmt.Errorf("missing create greptimedb cluster options")
	}
	clusterOpt := options.Cluster
	resourceName, resourceNamespace := options.Name, options.Namespace

	if clusterOpt.UseGreptimeCNArtifacts && len(clusterOpt.ImageRegistry) == 0 {
		clusterOpt.ConfigValues += fmt.Sprintf("image.registry=%s,initializer.registry=%s,", AliCloudRegistry, AliCloudRegistry)
	}

	opts := &helm.LoadOptions{
		ReleaseName:   resourceName,
		Namespace:     resourceNamespace,
		ChartName:     artifacts.GreptimeDBClusterChartName,
		ChartVersion:  clusterOpt.GreptimeDBChartVersion,
		FromCNRegion:  clusterOpt.UseGreptimeCNArtifacts,
		ValuesOptions: *clusterOpt,
		EnableCache:   true,
		ValuesFile:    clusterOpt.ValuesFile,
	}
	manifests, err := c.helmLoader.LoadAndRenderChart(ctx, opts)
	if err != nil {
		return err
	}

	if c.dryRun {
		c.logger.V(0).Info(string(manifests))
		return nil
	}

	if err = c.client.Apply(ctx, manifests); err != nil {
		return err
	}

	return c.client.WaitForClusterReady(ctx, resourceName, resourceNamespace, c.timeout)
}

// createEtcdCluster creates Etcd cluster.
func (c *Cluster) createEtcdCluster(ctx context.Context, options *opt.CreateOptions) error {
	if options.Etcd == nil {
		return fmt.Errorf("missing create etcd cluster options")
	}
	etcdOpt := options.Etcd
	resourceName, resourceNamespace := EtcdClusterName(options.Name), options.Namespace

	etcdOpt.ConfigValues += disableRBACConfig
	if etcdOpt.UseGreptimeCNArtifacts && len(etcdOpt.ImageRegistry) == 0 {
		etcdOpt.ConfigValues += fmt.Sprintf("image.registry=%s,", AliCloudRegistry)
	}

	opts := &helm.LoadOptions{
		ReleaseName:   resourceName,
		Namespace:     resourceNamespace,
		ChartName:     artifacts.EtcdChartName,
		ChartVersion:  artifacts.DefaultEtcdChartVersion,
		FromCNRegion:  etcdOpt.UseGreptimeCNArtifacts,
		ValuesOptions: *etcdOpt,
		EnableCache:   true,
		ValuesFile:    etcdOpt.ValuesFile,
	}
	manifests, err := c.helmLoader.LoadAndRenderChart(ctx, opts)
	if err != nil {
		return fmt.Errorf("error while loading helm chart: %v", err)
	}

	if c.dryRun {
		c.logger.V(0).Info(string(manifests))
		return nil
	}

	if err = c.client.Apply(ctx, manifests); err != nil {
		return fmt.Errorf("error while applying helm chart: %v", err)
	}

	return c.client.WaitForEtcdReady(ctx, resourceName, resourceNamespace, c.timeout)
}

func EtcdClusterName(clusterName string) string {
	return fmt.Sprintf("%s-etcd", clusterName)
}

func OperatorName() string {
	return "greptimedb-operator"
}
