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

package util

import (
	"testing"
)

func TestDetectQueryIntent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected QueryIntent
	}{
		{
			name:  "SQL查询语句识别",
			input: "用select node,used*rand(3)/total,timestamp from `lindb.monitor.system.mem_stat`来查一下 test44的数据",
			expected: QueryIntent{
				HasQueryIntent:  true,
				DataSourceName: "test44",
				SQLQuery:       "select node,used*rand(3)/total,timestamp from `lindb.monitor.system.mem_stat`",
				IsExplicitQuery: true,
			},
		},
		{
			name:  "中文查询关键词识别",
			input: "查询test44的数据",
			expected: QueryIntent{
				HasQueryIntent:  true,
				DataSourceName: "test44",
				SQLQuery:       "",
				IsExplicitQuery: true,
			},
		},
		{
			name:  "英文查询关键词识别",
			input: "select * from metrics where host='server1'",
			expected: QueryIntent{
				HasQueryIntent:  true,
				DataSourceName: "",
				SQLQuery:       "select * from metrics where host='server1'",
				IsExplicitQuery: true,
			},
		},
		{
			name:  "无查询意图",
			input: "你好，今天天气怎么样？",
			expected: QueryIntent{
				HasQueryIntent:  false,
				DataSourceName: "",
				SQLQuery:       "",
				IsExplicitQuery: false,
			},
		},
		{
			name:  "数据源名称提取",
			input: "查看数据源test44的指标",
			expected: QueryIntent{
				HasQueryIntent:  true,
				DataSourceName: "test44",
				SQLQuery:       "",
				IsExplicitQuery: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectQueryIntent(tt.input)

			if result.HasQueryIntent != tt.expected.HasQueryIntent {
				t.Errorf("HasQueryIntent: got %v, want %v", result.HasQueryIntent, tt.expected.HasQueryIntent)
			}

			if result.DataSourceName != tt.expected.DataSourceName {
				t.Errorf("DataSourceName: got %s, want %s", result.DataSourceName, tt.expected.DataSourceName)
			}

			if result.SQLQuery != tt.expected.SQLQuery {
				t.Errorf("SQLQuery: got %s, want %s", result.SQLQuery, tt.expected.SQLQuery)
			}

			if result.IsExplicitQuery != tt.expected.IsExplicitQuery {
				t.Errorf("IsExplicitQuery: got %v, want %v", result.IsExplicitQuery, tt.expected.IsExplicitQuery)
			}
		})
	}
}

func TestExtractSQLQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "标准SELECT语句",
			input:    "SELECT node, usage FROM metrics WHERE host='server1'",
			expected: "SELECT node, usage FROM metrics WHERE host='server1'",
		},
		{
			name:     "带反引号的表名",
			input:    "select node from `lindb.monitor.system.mem_stat`",
			expected: "select node from `lindb.monitor.system.mem_stat`",
		},
		{
			name:     "无SQL语句",
			input:    "今天天气不错",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSQLQuery(tt.input)
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestExtractDataSourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "标准数据源名称",
			input:    "查询test44的数据",
			expected: "test44",
		},
		{
			name:     "带数据源前缀",
			input:    "数据源test44的指标",
			expected: "test44",
		},
		{
			name:     "无数据源名称",
			input:    "查询数据",
			expected: "",
		},
		{
			name:     "无效数据源名称",
			input:    "查询数据源123",
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDataSourceName(tt.input)
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}