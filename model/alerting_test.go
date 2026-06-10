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

package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// TestAlertRule_ToDataQuery covers the three data formats that may be stored in the
// alert_rules.data column, each produced by a different frontend/client version:
//
//  1. Double-encoded string: the frontend JSON-stringified the queries array before
//     sending it, so the column value is a JSON string like `"[{\"refId\":...}]"`.
//  2. Array format: plain `[{...}]` without a QueryRequest wrapper.
//  3. Full QueryRequest object: `{"range":{...},"queries":[...]}`.
func TestAlertRule_ToDataQuery(t *testing.T) {
	cases := []struct {
		name       string
		data       datatypes.JSON // raw bytes stored in the column
		wantRefID  string
		wantErrMsg string // non-empty → expect error containing this substring
	}{
		{
			name: "double-encoded string wrapping a queries array",
			// Simulate: column = `"[{\"refId\":\"A\",\"datasource\":{\"uid\":\"ds1\"}}]"`
			data: datatypes.JSON(func() []byte {
				inner := `[{"refId":"A","datasource":{"uid":"ds1"}}]`
				b, _ := json.Marshal(inner) // produces `"[{\"refId\":\"A\",...}]"`
				return b
			}()),
			wantRefID: "A",
		},
		{
			name: "plain array format (no QueryRequest wrapper)",
			// Simulate: column = `[{"refId":"B","datasource":{"uid":"ds2"}}]`
			data:      datatypes.JSON(`[{"refId":"B","datasource":{"uid":"ds2"}}]`),
			wantRefID: "B",
		},
		{
			name: "full QueryRequest object",
			// Simulate: column = `{"range":{},"queries":[{"refId":"C","datasource":{"uid":"ds3"}}]}`
			data:      datatypes.JSON(`{"range":{},"queries":[{"refId":"C","datasource":{"uid":"ds3"}}]}`),
			wantRefID: "C",
		},
		{
			name:       "invalid JSON returns error",
			data:       datatypes.JSON(`not-json`),
			wantErrMsg: "err",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rule := &AlertRule{Data: tt.data}
			req, err := rule.ToDataQuery()
			if tt.wantErrMsg != "" {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, req)
			assert.Len(t, req.Queries, 1)
			assert.Equal(t, tt.wantRefID, req.Queries[0].RefID)
		})
	}
}
