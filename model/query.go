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
)

// TimeRange represents data query time range.
type TimeRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type QueryRequest struct {
	Range   TimeRange `json:"range"`
	Queries []*Query  `json:"queries"`
}

type Query struct {
	Datasource TargetDatasource `json:"datasource" binding:"required"`
	Request    json.RawMessage  `json:"request"`
	RefID      string           `json:"refId"`
}

type TargetDatasource struct {
	UID string `json:"uid" binding:"required"`
}

type QueryResponse struct {
	Results map[string]any `json:"results"`
}
