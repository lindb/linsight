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

package lindb

import (
	"bytes"
	"fmt"
)

// MetadataType represents metadata type for LinDB.
type MetadataType = string

// Defines all LinDB's metadata types.
var (
	Namespace MetadataType = "namespace"
	Metric                 = "metric"
	Field                  = "field"
	TagKey                 = "tagKey"
	TagValue               = "tagValue"
)

// Operator represents binary operator.
type Operator = string

var (
	Eq   Operator = "="
	In            = "in"
	GtEq          = ">="
	LtEq          = "<="
)

// DatasourceConfig represents datasource config for LinDB.
type DatasourceConfig struct {
	Database string `json:"database"`
}

// DataQueryRequest represents data query request for LinDB.
type DataQueryRequest struct {
	Namespace string   `json:"namespace"`
	Metric    string   `json:"metric"`
	Fields    []string `json:"fields"`
	GroupBy   []string `json:"groupBy"`
	Where     []Expr   `json:"where"`
	Stats     bool     `json:"stats"`
}

// MetadataQueryRequest represents metadata query request for LinDB.
type MetadataQueryRequest struct {
	Type      MetadataType `json:"type"`
	Prefix    string       `json:"prefix"`
	Namespace string       `json:"namespace"`
	Metric    string       `json:"metric"`
	TagKey    string       `json:"tagKey"`
}

// Expr represents where condition express.
type Expr struct {
	Key   string   `json:"key"`
	Op    Operator `json:"operator"`
	Value any      `json:"value"`
}

// String returns the string value of expr.
func (e Expr) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString(e.Key)
	fmt.Fprintf(buf, " %s ", e.Op)
	switch e.Value.(type) {
	case string:
		fmt.Fprintf(buf, "%v", e.Value)
	default:
		fmt.Fprintf(buf, "%v", e.Value)
	}
	return buf.String()
}
