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

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
)

func (c *Cluster) Delete(ctx context.Context, options *opt.DeleteOptions) error {
	// TODO(sh2): check whether the cluster is still running
	if err := c.delete(ctx, options); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) delete(ctx context.Context, options *opt.DeleteOptions) error {
	if err := c.cc.Frontend.Delete(ctx, options); err != nil {
		return err
	}
	if err := c.cc.Datanode.Delete(ctx, options); err != nil {
		return err
	}
	if err := c.cc.MetaSrv.Delete(ctx, options); err != nil {
		return err
	}
	if err := c.cc.Etcd.Delete(ctx, options); err != nil {
		return err
	}

	return nil
}
