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

package deployer

import (
	"context"
)

// Deployer is the general interface to handle the deployment of GreptimeDB cluster in different envrionment.
type Deployer interface {
	// GetGreptimeDBCluster get the current deployed GreptimeDBCluster by its name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error)

	// ListGreptimeDBClusters list all the deployed GreptimeDBClusters.
	ListGreptimeDBClusters(ctx context.Context, options *ListGreptimeDBClustersOptions) ([]*GreptimeDBCluster, error)

	// CreateGreptimeDBCluster create a GreptimeDBCluster with the given cluster name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	CreateGreptimeDBCluster(ctx context.Context, name string, options *CreateGreptimeDBClusterOptions) error

	// UpdateGreptimeDBCluster update the GreptimeDBCluster spec with the given cluster name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	UpdateGreptimeDBCluster(ctx context.Context, name string, options *UpdateGreptimeDBClusterOptions) error

	// DeleteGreptimeDBCluster delete the GreptimeDBCluster with the given cluster name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	DeleteGreptimeDBCluster(ctx context.Context, name string, options *DeleteGreptimeDBClusterOption) error

	// CreateEtcdCluster create etcd cluster with the given cluster name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	CreateEtcdCluster(ctx context.Context, name string, options *CreateEtcdClusterOptions) error

	// DeleteEtcdCluster delete the etcd cluster with the given cluster name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error

	// CreateGreptimeDBOperator create a GreptimeDBOperator with the given operator name.
	// The name is the namespaced name(<namespace>/<name>) in Kubernetes.
	// The API only works for Kubernetes.
	CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error
}

// GreptimeDBCluster is the internal type of gtctl to describe GreptimeDB cluster.
// We want to make the Depolyer decouple from K8s or any other specified envioronment.
type GreptimeDBCluster struct {
	// Raw can be *greptimedbclusterv1alpha1.GreptimeDBCluster.
	Raw interface{}
}

// GetGreptimeDBClusterOptions is the options to get a GreptimeDB cluster.
type GetGreptimeDBClusterOptions struct{}

// ListGreptimeDBClustersOptions is the options to list all the GreptimeDB clusters.
type ListGreptimeDBClustersOptions struct{}

// CreateGreptimeDBClusterOptions is the options to create a GreptimeDB cluster.
type CreateGreptimeDBClusterOptions struct {
	GreptimeDBChartVersion string

	ImageRegistry               string `helm:"image.registry"`
	DatanodeStorageClassName    string `helm:"datanode.storage.storageClassName"`
	DatanodeStorageSize         string `helm:"datanode.storage.storageSize"`
	DatanodeStorageRetainPolicy string `helm:"datanode.storage.storageRetainPolicy"`
	EtcdEndPoint                string `helm:"etcdEndpoints"`
}

// UpdateGreptimeDBClusterOptions is the options to update a GreptimeDB cluster.
type UpdateGreptimeDBClusterOptions struct {
	NewCluster *GreptimeDBCluster
}

// DeleteGreptimeDBClusterOption is the options to delete a GreptimeDB cluster.
type DeleteGreptimeDBClusterOption struct{}

// CreateEtcdClusterOptions is the options to create an etcd cluster.
type CreateEtcdClusterOptions struct {
	EtcdChartVersion string

	ImageRegistry        string `helm:"image.registry"`
	EtcdStorageClassName string `helm:"storage.storageClassName"`
	EtcdStorageSize      string `helm:"storage.volumeSize"`
}

// DeleteEtcdClusterOption is the options to delete an etcd cluster.
type DeleteEtcdClusterOption struct{}

// CreateGreptimeDBOperatorOptions is the options to create a GreptimeDB operator.
type CreateGreptimeDBOperatorOptions struct {
	GreptimeDBOperatorChartVersion string

	ImageRegistry string `helm:"image.registry"`
}
