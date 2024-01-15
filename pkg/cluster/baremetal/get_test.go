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

package baremetal

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectPidsForBareMetal(t *testing.T) {
	pidsPath := filepath.Join("testdata", "pids")
	want := map[string]string{
		"a": "123",
		"b": "456",
		"c": "789",
	}

	ret := collectPidsForBareMetal(pidsPath)

	assert.Equal(t, want, ret)
}
