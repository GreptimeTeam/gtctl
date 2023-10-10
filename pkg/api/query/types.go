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

package query

import (
	"context"

	"github.com/olekukonko/tablewriter"
)

// Getter defines the get operation for one cluster.
type Getter interface {
	// Get gets the current cluster profile.
	Get(ctx context.Context, options *Options) error
}

// Lister defines the list operation for one cluster.
type Lister interface {
	// List lists the current cluster profiles.
	List(ctx context.Context, options *Options) error
}

type Options struct {
	Namespace string
	Name      string

	// Table view render
	Table *tablewriter.Table
}
