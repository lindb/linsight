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

package tools

import (
	"encoding/json"
)

// ToolCallResult is the universal return shape for all builtin tools.
// Data is the LLM-visible payload; Extra carries runtime-generated metadata
// (e.g. row_count, count) that flows to SSE events and DB persistence
// but is hidden from the LLM context.
type ToolCallResult struct {
	Data  json.RawMessage `json:"data"`
	Extra map[string]any  `json:"extra,omitempty"`
}

// ParseToolCallResult attempts to unmarshal s as a ToolCallResult.
// Returns (result, true) when s is valid ToolCallResult JSON with a non-empty Data field.
// Returns (zero, false) for plain-text output, arbitrary JSON, or empty strings
// (e.g. MCP tool responses that are not ToolCallResult).
func ParseToolCallResult(s string) (ToolCallResult, bool) {
	if s == "" {
		return ToolCallResult{}, false
	}
	var tcr ToolCallResult
	if err := json.Unmarshal([]byte(s), &tcr); err != nil {
		return ToolCallResult{}, false
	}
	// Require a non-empty Data field to distinguish ToolCallResult from arbitrary JSON
	// objects that happen to have an "extra" key but are not produced by builtin tools.
	if len(tcr.Data) == 0 {
		return ToolCallResult{}, false
	}
	return tcr, true
}
