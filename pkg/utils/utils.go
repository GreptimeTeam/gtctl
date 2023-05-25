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

package utils

import (
	"fmt"
	"os"
)

func CreateDirIfNotExists(dir string) (err error) {
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func IsFileExists(filepath string) (bool, error) {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		// file does not exist
		return false, nil
	}

	if err != nil {
		// Other errors happened.
		return false, err
	}

	if info.IsDir() {
		// It's a directory.
		return false, fmt.Errorf("'%s' is directory, not file", filepath)
	}

	// The file exists.
	return true, nil
}
