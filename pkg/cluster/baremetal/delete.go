/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package baremetal

import (
	"context"
	"fmt"
	"os"
	"syscall"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

func (c *Cluster) Delete(ctx context.Context, options *opt.DeleteOptions) error {
	cluster, err := c.get(ctx, &opt.GetOptions{Name: options.Name})
	if err != nil {
		return err
	}

	running, ferr, serr := c.isClusterRunning(cluster.ForegroundPid)
	if ferr != nil {
		return fmt.Errorf("error checking whether cluster '%s' is running: %v", options.Name, ferr)
	}
	if running || serr == nil {
		return fmt.Errorf("cluster '%s' is running, please stop it before deleting", options.Name)
	}

	csd := c.mm.GetClusterScopeDirs()
	c.logger.V(0).Infof("Deleting cluster configurations and runtime directories in %s", csd.BaseDir)
	if err = c.delete(ctx, csd.BaseDir); err != nil {
		return err
	}
	c.logger.V(0).Info("Deleted!")

	return nil
}

func (c *Cluster) delete(_ context.Context, baseDir string) error {
	return fileutils.DeleteDirIfExists(baseDir)
}

// isClusterRunning checks the current status of cluster by sending signal to process.
func (c *Cluster) isClusterRunning(pid int) (runs bool, f error, s error) {
	p, f := os.FindProcess(pid)
	if f != nil {
		return false, f, nil
	}

	s = p.Signal(syscall.Signal(0))
	if s != nil {
		return false, nil, s
	}

	return true, nil, nil
}
