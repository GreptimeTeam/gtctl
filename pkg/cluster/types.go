// Copyright 2024 Greptime Team
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

package cluster

import (
	"context"

	greptimedbclusterv1alpha1 "github.com/GreptimeTeam/greptimedb-operator/apis/v1alpha1"
	"github.com/olekukonko/tablewriter"

	"github.com/GreptimeTeam/gtctl/pkg/status"
)

type Operations interface {
	// Get gets the current cluster profile.
	Get(ctx context.Context, options *GetOptions) error

	// List lists all cluster profiles.
	List(ctx context.Context, options *ListOptions) error

	// Scale scales the current cluster according to NewReplicas in ScaleOptions,
	// and refill the OldReplicas in ScaleOptions.
	Scale(ctx context.Context, options *ScaleOptions) error

	// Create creates a new cluster.
	Create(ctx context.Context, options *CreateOptions) error

	// Delete deletes a specific cluster.
	Delete(ctx context.Context, options *DeleteOptions) error

	// Connect connects to a specific cluster.
	Connect(ctx context.Context, options *ConnectOptions) error
}

type GetOptions struct {
	Namespace string
	Name      string

	// Table view render.
	Table *tablewriter.Table
}

type ListOptions struct {
	GetOptions
}

type ScaleOptions struct {
	NewReplicas   int32
	OldReplicas   int32
	Namespace     string
	Name          string
	ComponentType greptimedbclusterv1alpha1.ComponentKind
}

type DeleteOptions struct {
	Namespace    string
	Name         string
	TearDownEtcd bool
}

type CreateOptions struct {
	Namespace string
	Name      string

	Cluster  *CreateClusterOptions
	Operator *CreateOperatorOptions
	Etcd     *CreateEtcdOptions

	Spinner *status.Spinner
}

// CreateClusterOptions is the options to create a GreptimeDB cluster.
type CreateClusterOptions struct {
	GreptimeDBChartVersion string
	UseGreptimeCNArtifacts bool
	ValuesFile             string

	ImageRegistry               string `helm:"image.registry"`
	InitializerImageRegistry    string `helm:"initializer.registry"`
	DatanodeStorageClassName    string `helm:"datanode.storage.storageClassName"`
	DatanodeStorageSize         string `helm:"datanode.storage.storageSize"`
	DatanodeStorageRetainPolicy string `helm:"datanode.storage.storageRetainPolicy"`
	EtcdEndPoints               string `helm:"meta.etcdEndpoints"`
	ConfigValues                string `helm:"*"`
}

// CreateOperatorOptions is the options to create a GreptimeDB operator.
type CreateOperatorOptions struct {
	GreptimeDBOperatorChartVersion string
	UseGreptimeCNArtifacts         bool
	ValuesFile                     string

	ImageRegistry string `helm:"image.registry"`
	ConfigValues  string `helm:"*"`
}

// CreateEtcdOptions is the options to create an etcd cluster.
type CreateEtcdOptions struct {
	EtcdChartVersion       string
	UseGreptimeCNArtifacts bool
	ValuesFile             string

	// The parameters reference: https://artifacthub.io/packages/helm/bitnami/etcd.
	EtcdClusterSize      string `helm:"replicaCount"`
	ImageRegistry        string `helm:"image.registry"`
	EtcdStorageClassName string `helm:"persistence.storageClass"`
	EtcdStorageSize      string `helm:"persistence.size"`
	ConfigValues         string `helm:"*"`
}

type ConnectProtocol int

const (
	MySQL ConnectProtocol = iota
	Postgres
)

type ConnectOptions struct {
	Namespace string
	Name      string
	Protocol  ConnectProtocol
}
