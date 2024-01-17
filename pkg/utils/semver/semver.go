// Copyright 2023 Greptime Team
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
	"github.com/Masterminds/semver/v3"
)

// Compare compares two semantic versions.
// It returns true if v1 is greater than v2, otherwise false.
func Compare(v1, v2 string) (bool, error) {
	semV1, err := semver.NewVersion(v1)
	if err != nil {
		return false, err
	}

	semV2, err := semver.NewVersion(v2)
	if err != nil {
		return false, err
	}

	return semV1.GreaterThan(semV2), nil
}
