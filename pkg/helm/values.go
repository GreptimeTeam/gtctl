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

package helm

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"
)

const (
	FieldTag = "helm"
)

// Values is a map of helm values.
type Values map[string]interface{}

// NewFromFile creates a new Values from a local yaml values file.
func NewFromFile(filename string) (Values, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var values Values
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}

	return values, nil
}

// ToHelmValues converts the input struct that contains special helm annotations `helm:"values"` and the local yaml values file to a map that can be used as helm values.
// If there is the same key in both the input struct and the local yaml values file, the value in the input struct will be used.
// valuesFile can be empty.
func ToHelmValues(input interface{}, valuesFile string) (Values, error) {
	var (
		base Values
		err  error
	)

	if len(valuesFile) > 0 {
		base, err = NewFromFile(valuesFile)
		if err != nil {
			return nil, err
		}
	}

	vals, err := struct2Values(input)
	if err != nil {
		return nil, err
	}

	return mergeMaps(base, vals), nil
}

// OutputValues returns the values as a yaml byte array.
func (v Values) OutputValues() ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// mergeMaps merges two maps recursively.
// The function is copied from 'helm/helm/pkg/cli/values/options.go'.
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// struct2Values converts the input struct to a helm values map.
func struct2Values(input interface{}) (Values, error) {
	var rawArgs []string
	valueOf := reflect.ValueOf(input)

	// Make sure we are handling with a struct here.
	if valueOf.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid input type, should be struct")
	}

	typeOf := reflect.TypeOf(input)
	for i := 0; i < valueOf.NumField(); i++ {
		helmValueKey := typeOf.Field(i).Tag.Get(FieldTag)
		if len(helmValueKey) > 0 && valueOf.Field(i).Len() > 0 {
			// If the struct annotation is `helm:"*"`, the value will be added to the rawArgs directly.
			// Otherwise, the value will be added to the rawArgs with the key `helmValueKey`.
			if helmValueKey == "*" {
				rawArgs = append(rawArgs, valueOf.Field(i).String())
			} else {
				rawArgs = append(rawArgs, fmt.Sprintf("%s=%s", helmValueKey, valueOf.Field(i)))
			}
		}
	}

	if len(rawArgs) > 0 {
		values := make(map[string]interface{})
		if err := strvals.ParseInto(strings.Join(rawArgs, ","), values); err != nil {
			return nil, err
		}
		return values, nil
	}

	return nil, nil
}
