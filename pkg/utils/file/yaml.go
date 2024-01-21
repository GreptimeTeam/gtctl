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

package file

import (
	"bytes"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

// MergeYAML merges two yaml files from src to dst, the src yaml will override dst yaml if the same key exists.
func MergeYAML(dst, src []byte) ([]byte, error) {
	map1 := map[string]interface{}{}
	map2 := map[string]interface{}{}

	if err := yaml.Unmarshal(src, &map1); err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(dst, &map2); err != nil {
		return nil, err
	}

	maps.Copy(map2, map1)

	buf := bytes.NewBuffer([]byte{})
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(map2); err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
