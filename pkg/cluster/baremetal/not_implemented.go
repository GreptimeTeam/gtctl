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

package baremetal

import (
	"context"
	"fmt"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/status"
)

func (c *Cluster) List(ctx context.Context, options *opt.ListOptions) error {
	return fmt.Errorf("do not support")
}

func (c *Cluster) Scale(ctx context.Context, options *opt.ScaleOptions) error {
	return fmt.Errorf("do not support")
}

func (c *Cluster) Create(ctx context.Context, options *opt.CreateOptions, spinner *status.Spinner) error {
	return fmt.Errorf("do not support")
}
