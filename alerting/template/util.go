// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package template

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// parseJSONMap decodes a datatypes.JSON field into a string map.
func parseJSONMap(raw datatypes.JSON) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	result := make(map[string]string)
	_ = json.Unmarshal(raw, &result)
	return result
}

// parseJSONMapFromAny decodes an interface{} that may be a datatypes.JSON, []byte, or map.
func parseJSONMapFromAny(v interface{}) map[string]string {
	switch t := v.(type) {
	case datatypes.JSON:
		return parseJSONMap(t)
	case []byte:
		result := make(map[string]string)
		_ = json.Unmarshal(t, &result)
		return result
	case map[string]string:
		return t
	case map[string]interface{}:
		result := make(map[string]string, len(t))
		for k, val := range t {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result
	default:
		return nil
	}
}
