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

package components

import (
	"context"

	opt "github.com/GreptimeTeam/gtctl/pkg/cluster"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
)

const (
	DefaultLogLevel = "info"
)

// WorkingDirs include all the dirs used in bare-metal mode.
type WorkingDirs struct {
	DataDir string `yaml:"dataDir"`
	LogsDir string `yaml:"logsDir"`
	PidsDir string `yaml:"pidsDir"`
}

type allocatedDirs struct {
	dataDirs []string
	logsDirs []string
	pidsDirs []string
}

func (ad *allocatedDirs) delete(_ context.Context, _ *opt.DeleteOptions) error {
	for _, dir := range ad.logsDirs {
		if err := fileutils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	for _, dir := range ad.dataDirs {
		if err := fileutils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	for _, dir := range ad.pidsDirs {
		if err := fileutils.DeleteDirIfExists(dir); err != nil {
			return err
		}
	}

	return nil
}

// ClusterComponent is the basic component of running GreptimeDB Cluster in bare-metal mode.
type ClusterComponent interface {
	// Start starts cluster component by executing binary.
	Start(ctx context.Context, stop context.CancelFunc, binary string) error

	// BuildArgs build up args for cluster component.
	BuildArgs(params ...interface{}) []string

	// IsRunning returns the status of current cluster component.
	IsRunning(ctx context.Context) bool

	// Delete deletes resources that allocated in the system for current component.
	Delete(ctx context.Context, options *opt.DeleteOptions) error

	// Name return the name of component.
	Name() string
}
