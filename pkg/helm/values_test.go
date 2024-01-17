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

package helm

import (
	"os"
	"testing"
)

func TestNewFromFile(t *testing.T) {
	v, err := NewFromFile("testdata/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	output, err := v.OutputValues()
	if err != nil {
		t.Fatal(err)
	}

	original, err := os.ReadFile("testdata/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if string(original) != string(output) {
		t.Errorf("expected %s, got %s", string(original), string(output))
	}
}

func TestToHelmValues(t *testing.T) {
	inputVals := struct {
		ImageRegistry string `helm:"image.registry"`
		Version       string `helm:"image.tag"`
		ConfigValues  string `helm:"*"`
	}{
		ImageRegistry: "greptime-registry.cn-hangzhou.cr.aliyuncs.com",
		Version:       "v0.1.0",
		ConfigValues:  "resources.limits.cpu=100m,resources.limits.memory=256Mi",
	}

	v, err := ToHelmValues(inputVals, "testdata/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	output, err := v.OutputValues()
	if err != nil {
		t.Fatal(err)
	}

	original, err := os.ReadFile("testdata/merged-values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if string(original) != string(output) {
		t.Errorf("expected %s, got %s", string(original), string(output))
	}
}
