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
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/GreptimeTeam/gtctl/pkg/artifacts"
	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

func (c *Cluster) Create(ctx context.Context, options *opt.CreateOptions) error {
	spinner := options.Spinner

	withSpinner := func(target string, f func(context.Context, *opt.CreateOptions) error) error {
		if spinner != nil {
			spinner.Start(fmt.Sprintf("Installing %s...", target))
		}

		if err := f(ctx, options); err != nil {
			if spinner != nil {
				spinner.Stop(false, fmt.Sprintf("Installing %s failed", target))
			}
			return err
		}

		if spinner != nil {
			spinner.Stop(true, fmt.Sprintf("Installing %s successfully 🎉", target))
		}
		return nil
	}

	if err := withSpinner("Etcd Cluster", c.createEtcdCluster); err != nil {
		return err
	}
	if err := withSpinner("GreptimeDB Cluster", c.createCluster); err != nil {
		if err := c.Wait(ctx, true); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (c *Cluster) createCluster(ctx context.Context, options *opt.CreateOptions) error {
	if options.Cluster == nil {
		return fmt.Errorf("missing create greptimedb cluster options")
	}
	clusterOpt := options.Cluster

	var binPath string
	if c.config.Cluster.Artifact != nil {
		if c.config.Cluster.Artifact.Local != "" {
			binPath = c.config.Cluster.Artifact.Local

			// Ensure the binary path exists.
			if exist, _ := fileutils.IsFileExists(binPath); !exist {
				return fmt.Errorf("greptimedb cluster artifact '%s' is not exist", binPath)
			}
		} else {
			src, err := c.am.NewSource(artifacts.GreptimeBinName, c.config.Cluster.Artifact.Version,
				artifacts.ArtifactTypeBinary, clusterOpt.UseGreptimeCNArtifacts)
			if err != nil {
				return err
			}

			destDir, err := c.mm.AllocateArtifactFilePath(src, false)
			if err != nil {
				return err
			}

			installDir, err := c.mm.AllocateArtifactFilePath(src, true)
			if err != nil {
				return err
			}

			artifactFile, err := c.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{
				EnableCache:      c.enableCache,
				BinaryInstallDir: installDir,
			})
			if err != nil {
				return err
			}
			binPath = artifactFile
		}
	}

	if err := c.cc.MetaSrv.Start(c.ctx, c.stop, binPath); err != nil {
		return err
	}
	if err := c.cc.Datanode.Start(c.ctx, c.stop, binPath); err != nil {
		return err
	}
	if err := c.cc.Frontend.Start(c.ctx, c.stop, binPath); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) createEtcdCluster(ctx context.Context, options *opt.CreateOptions) error {
	if options.Etcd == nil {
		return fmt.Errorf("missing create etcd cluster options")
	}
	etcdOpt := options.Etcd

	var binPath string
	if c.config.Etcd.Artifact != nil {
		if c.config.Etcd.Artifact.Local != "" {
			binPath = c.config.Etcd.Artifact.Local

			// Ensure the binary path exists.
			if exist, _ := fileutils.IsFileExists(binPath); !exist {
				return fmt.Errorf("etcd artifact '%s' is not exist", binPath)
			}
		} else {
			src, err := c.am.NewSource(artifacts.EtcdBinName, c.config.Etcd.Artifact.Version,
				artifacts.ArtifactTypeBinary, etcdOpt.UseGreptimeCNArtifacts)
			if err != nil {
				return err
			}

			destDir, err := c.mm.AllocateArtifactFilePath(src, false)
			if err != nil {
				return err
			}

			installDir, err := c.mm.AllocateArtifactFilePath(src, true)
			if err != nil {
				return err
			}

			artifactFile, err := c.am.DownloadTo(ctx, src, destDir, &artifacts.DownloadOptions{
				EnableCache:      c.enableCache,
				BinaryInstallDir: installDir,
			})
			if err != nil {
				return err
			}
			binPath = artifactFile
		}
	}

	if err := c.cc.Etcd.Start(c.ctx, c.stop, binPath); err != nil {
		return err
	}
	if err := c.checkEtcdHealth(binPath); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) checkEtcdHealth(etcdBin string) error {
	// It's very likely that "etcdctl" is under the same directory of "etcd".
	etcdctlBin := path.Join(etcdBin, "../etcdctl")
	exists, err := fileutils.IsFileExists(etcdctlBin)
	if err != nil {
		return err
	}
	if !exists {
		c.logger.V(3).Infof("'etcdctl' is not found under the same directory of 'etcd', skip checking the healthy of Etcd.")
		return nil
	}

	for retry := 0; retry < 10; retry++ {
		outputRaw, err := exec.Command(etcdctlBin, "endpoint", "status").Output()
		if err != nil {
			return err
		}
		output := string(outputRaw)
		statuses := strings.Split(output, "\n")

		hasLeader := false
		for i := 0; i < len(statuses); i++ {
			fields := strings.Split(statuses[i], ",")

			// We are checking Etcd status with default output format("--write-out=simple"), example output:
			// 127.0.0.1:2379, 8e9e05c52164694d, 3.5.0, 131 kB, true, false, 3, 75, 75,
			//
			// The output fields are corresponding to the following table's columns (with format "--write-out=table"):
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			// |    ENDPOINT    |        ID        | VERSION | DB SIZE | IS LEADER | IS LEARNER | RAFT TERM | RAFT INDEX | RAFT APPLIED INDEX | ERRORS |
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			// | 127.0.0.1:2379 | 8e9e05c52164694d |   3.5.0 |  131 kB |      true |      false |         3 |         72 |                 72 |        |
			// +----------------+------------------+---------+---------+-----------+------------+-----------+------------+--------------------+--------+
			//
			// So we can just check the "IS LEADER" field.
			if strings.TrimSpace(fields[4]) == "true" {
				hasLeader = true
				break
			}
		}
		if hasLeader {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	csd := c.mm.GetClusterScopeDirs()
	return fmt.Errorf("etcd is not ready in 10 second! You can find its logs in %s", path.Join(csd.LogsDir, "etcd"))
}

func (c *Cluster) Wait(ctx context.Context, close bool) error {
	v := c.config.Cluster.Artifact.Version
	if len(v) == 0 {
		v = "unknown"
	}

	csd := c.mm.GetClusterScopeDirs()
	if !close {
		c.logger.V(0).Infof("The cluster(pid=%d, version=%s) is running in bare-metal mode now...", os.Getpid(), v)
		c.logger.V(0).Infof("To view dashboard by accessing: %s", logger.Bold("http://localhost:4000/dashboard/"))
	} else {
		c.logger.Warnf("The cluster(pid=%d, version=%s) run in bare-metal has been shutting down...", os.Getpid(), v)
		c.logger.Warnf("To view the failure by browsing logs in: %s", logger.Bold(csd.LogsDir))
		return nil
	}

	// Wait for all the sub-processes to exit.
	if err := c.wait(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Cluster) wait(_ context.Context) error {
	c.wg.Wait()

	// We ignore the context from input params, since
	// it is not the context of current cluster.
	<-c.ctx.Done()

	csd := c.mm.GetClusterScopeDirs()
	c.logger.V(0).Infof("Cluster is shutting down, don't worry, it still remain in %s", logger.Bold(csd.BaseDir))
	return nil
}
