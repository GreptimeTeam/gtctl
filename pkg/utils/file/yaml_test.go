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

package file

import (
	"reflect"
	"testing"
)

func TestMergeYAML(t *testing.T) {
	tests := []struct {
		name  string
		yaml1 string
		yaml2 string
		want  string
	}{
		{
			name: "test",
			yaml1: `
a: a
b: 
  c:
    d: d
e:
  f:
    - g
k:
  l: l
`,
			yaml2: `
a: a1
b: 
  c:
    d: d1
e:
  f:
    - h
i:
  j: j
`,
			want: `a: a1
b:
  c:
    d: d1
e:
  f:
    - h
i:
  j: j
k:
  l: l
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeYAML([]byte(tt.yaml1), []byte(tt.yaml2))
			if err != nil {
				t.Errorf("MergeYAML() err = %v", err)
			}

			actual := string(got)
			if !reflect.DeepEqual(actual, tt.want) {
				t.Errorf("MergeYAML() got = %v, want %v", actual, tt.want)
			}
		})
	}
}
