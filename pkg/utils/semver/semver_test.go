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

package semver

import (
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   bool
	}{
		{"v0.3.2", "v0.4.0-nightly-20230802", false},
		{"v0.4.0-nightly-20230807", "0.4.0-nightly-20230802", true},
	}

	for _, test := range tests {
		got, err := Compare(test.v1, test.v2)
		if err != nil {
			t.Errorf("compare '%s' and '%s': %v", test.v1, test.v2, err)
		}

		if got != test.want {
			t.Errorf("compare '%s' and '%s': got %v, want %v", test.v1, test.v2, got, test.want)
		}
	}
}
