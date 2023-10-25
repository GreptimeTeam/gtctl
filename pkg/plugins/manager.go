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

package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// DefaultPluginPrefix is the default prefix for the plugin binary name.
	DefaultPluginPrefix = "gtctl-"

	// PluginSearchPathsEnvKey is the environment variable key for the plugin search paths.
	PluginSearchPathsEnvKey = "GTCTL_PLUGIN_PATHS"
)

// Manager manages and executes the plugins.
type Manager struct {
	prefix      string
	searchPaths []string
}

func NewManager() (*Manager, error) {
	// Always search the current working directory.
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	m := &Manager{
		prefix:      DefaultPluginPrefix,
		searchPaths: []string{pwd},
	}

	pluginSearchPaths := os.Getenv(PluginSearchPathsEnvKey)
	if len(pluginSearchPaths) > 0 {
		m.searchPaths = append(strings.Split(pluginSearchPaths, ":"), m.searchPaths...)
	}

	return m, nil
}

// ShouldRun returns true whether you should run the plugin.
func (m *Manager) ShouldRun(err error) bool {
	// The error is returned by cobra itself.
	return strings.Contains(err.Error(), "unknown command")
}

// Run searches for the plugin and runs it.
func (m *Manager) Run(args []string) error {
	if len(args) < 1 {
		return nil // No arguments provided, normal help message will be shown.
	}

	pluginPath, err := m.searchPlugins(args[0])
	if err != nil {
		return err
	}

	pluginCmd := exec.Command(pluginPath, args[1:]...)
	pluginCmd.Stdin = os.Stdin
	pluginCmd.Stdout = os.Stdout
	pluginCmd.Stderr = os.Stderr
	if err := pluginCmd.Run(); err != nil {
		return fmt.Errorf("failed to run plugin '%s': %v", pluginPath, err)
	}

	return nil
}

func (m *Manager) searchPlugins(name string) (string, error) {
	if len(m.searchPaths) == 0 {
		return "", fmt.Errorf("no plugin search paths provided")
	}

	// Construct plugin binary name gtctl-<subcmd>.
	pluginName := m.prefix + name
	for _, path := range m.searchPaths {
		pluginPath := filepath.Join(path, pluginName)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			continue
		}

		return pluginPath, nil
	}

	return "", fmt.Errorf("error: unknown command %q for \"gtctl\"", name)
}
