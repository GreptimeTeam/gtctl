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
	"os"
	"testing"

	"sigs.k8s.io/kind/pkg/log"

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

const (
	testDir = "/tmp/gtctl-test-am"
)

func TestArtifactManager(t *testing.T) {
	am, err := NewArtifactManager(testDir, logger.New(os.Stdout, log.Level(4), logger.WithColored()), false)
	if err != nil {
		t.Errorf("failed to create artifact manager: %v", err)
	}

	// Cleanup test directory.
	defer func() {
		os.RemoveAll(testDir)
	}()

	testConfigs := []*config.Artifact{
		{
			Version: "latest",
		},
		{
			Version: BreakingChangeVersion,
		},
	}

	ctx := context.Background()
	for _, tc := range testConfigs {
		if err := am.PrepareArtifact(ctx, GreptimeArtifactType, tc); err != nil {
			t.Errorf("failed to prepare artifact: %v", err)
		}

		_, err := am.BinaryPath(GreptimeArtifactType, tc)
		if err != nil {
			t.Errorf("failed to get binary path: %v", err)
		}
	}
}
