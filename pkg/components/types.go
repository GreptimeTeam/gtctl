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
)

const (
	DefaultLogLevel = "info"
)

// WorkingDirs include all the directories used in bare-metal mode.
type WorkingDirs struct {
	DataDir string `yaml:"dataDir"`
	LogsDir string `yaml:"logsDir"`
	PidsDir string `yaml:"pidsDir"`
}

// allocatedDirs include all the directories that created during bare-metal mode.
type allocatedDirs struct {
	dataDirs []string
	logsDirs []string
	pidsDirs []string
}

// ClusterComponent is the basic component of running GreptimeDB Cluster in bare-metal mode.
type ClusterComponent interface {
	// Start starts cluster component by executing binary.
	Start(ctx context.Context, stop context.CancelFunc, binary string) error

	// BuildArgs build up args for cluster component.
	BuildArgs(params ...interface{}) []string

	// IsRunning returns the status of current cluster component.
	IsRunning(ctx context.Context) bool

	// Name return the name of component.
	Name() string
}
