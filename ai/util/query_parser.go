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
	"regexp"
	"strings"
)

// QueryIntent represents the detected query intent from user input
type QueryIntent struct {
	HasQueryIntent  bool   // Whether the input contains a query intent
	DataSourceName  string // Extracted data source name (e.g., "test44")
	SQLQuery        string // Extracted SQL query statement
	IsExplicitQuery bool   // Whether the user explicitly mentioned query keywords
}

// DetectQueryIntent analyzes user input to detect SQL query intent
func DetectQueryIntent(userInput string) *QueryIntent {
	intent := &QueryIntent{}

	// Normalize input for easier matching
	normalized := strings.ToLower(strings.TrimSpace(userInput))

	// Check for explicit query keywords
	queryKeywords := []string{
		"查询", "查一下", "查看", "select", "from", "where",
		"数据", "指标", "监控", "统计", "分析",
	}

	for _, keyword := range queryKeywords {
		if strings.Contains(normalized, keyword) {
			intent.IsExplicitQuery = true
			break
		}
	}

	// Extract SQL query using pattern matching
	intent.SQLQuery = extractSQLQuery(userInput)

	// Extract data source name
	intent.DataSourceName = extractDataSourceName(userInput)

	// Determine if there's a query intent
	intent.HasQueryIntent = intent.IsExplicitQuery || intent.SQLQuery != "" || intent.DataSourceName != ""

	return intent
}

// extractSQLQuery extracts SQL-like query statements from user input
func extractSQLQuery(input string) string {
	// Pattern for SQL SELECT statements - match from SELECT to end of statement
	sqlPattern := regexp.MustCompile(`(?i)(?:select\s+[^;]+?\s+from\s+[\w\d\.\`+"`"+`\-]+(?:\s+where\s+[^;]+?)?)`)

	matches := sqlPattern.FindStringSubmatch(input)
	if len(matches) > 0 {
		return strings.TrimSpace(matches[0])
	}

	return ""
}

// extractDataSourceName extracts data source names from user input
func extractDataSourceName(input string) string {
	// First, look for explicit mentions like "用test44的数据"
	explicitPattern := regexp.MustCompile(`(?i)(?:用|使用|查|查询)\s+([a-z0-9_-]+)(?:\s+的)?(?:数据|指标|监控)`)

	matches := explicitPattern.FindStringSubmatch(input)
	if len(matches) > 1 && isValidDataSourceName(matches[1]) {
		return matches[1]
	}

	// Then look for data source mentions (e.g., "test44", "数据源xxx")
	dsPattern := regexp.MustCompile(`(?i)(?:数据源\s*)?([a-z0-9_-]+)(?:\s+数据源)?`)

	matches = dsPattern.FindStringSubmatch(input)
	if len(matches) > 1 && isValidDataSourceName(matches[1]) {
		return matches[1]
	}

	// Finally, look for standalone data source names not part of SQL keywords
	// Avoid matching SQL keywords by checking context
	standalonePattern := regexp.MustCompile(`(?i)([a-z0-9_-]{2,50})(?:\s+的)?(?:数据|指标|监控)?`)

	matches = standalonePattern.FindStringSubmatch(input)
	if len(matches) > 1 && isValidDataSourceName(matches[1]) {
		// Check if it's not a SQL keyword and not preceded by SQL keywords
		if !isSQLKeyword(matches[1]) && !isPrecededBySQLKeyword(input, matches[0]) {
			return matches[1]
		}
	}

	return ""
}

// isSQLKeyword checks if a string is a common SQL keyword
func isSQLKeyword(s string) bool {
	sqlKeywords := map[string]bool{
		"select": true, "from": true, "where": true, "group": true, "order": true,
		"by": true, "having": true, "limit": true, "offset": true, "join": true,
		"inner": true, "outer": true, "left": true, "right": true, "full": true,
		"on": true, "as": true, "distinct": true, "and": true, "or": true,
		"not": true, "in": true, "like": true, "between": true, "is": true,
		"null": true, "true": true, "false": true,
	}
	return sqlKeywords[strings.ToLower(s)]
}

// isPrecededBySQLKeyword checks if a match is preceded by SQL keywords
func isPrecededBySQLKeyword(input, match string) bool {
	// Find the position of the match in the input
	matchIndex := strings.Index(strings.ToLower(input), strings.ToLower(match))
	if matchIndex == -1 {
		return false
	}

	// Check the text before the match for SQL keywords
	textBefore := input[:matchIndex]

	// Common SQL keywords that might precede a data source name
	sqlPrefixes := []string{
		"select", "from", "where", "group by", "order by",
		"having", "limit", "offset", "join", "on",
	}

	for _, prefix := range sqlPrefixes {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(textBefore)), prefix) {
			return true
		}
	}

	return false
}

// isValidDataSourceName checks if a string looks like a valid data source name
func isValidDataSourceName(name string) bool {
	if len(name) < 2 || len(name) > 50 {
		return false
	}

	// Basic validation: alphanumeric, hyphen, underscore
	validPattern := regexp.MustCompile(`^[a-z0-9_-]+$`)
	return validPattern.MatchString(strings.ToLower(name))
}