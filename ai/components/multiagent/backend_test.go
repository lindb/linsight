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

package multiagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSpecialist_FullFrontmatter(t *testing.T) {
	content := "---\n" +
		"description: Analyze metric anomalies for a service.\n" +
		"model: gpt-4o\n" +
		"tools: query, dashboards\n" +
		"---\n" +
		"You are a metrics analysis expert.\nAlways cite the datasource."

	rs, err := parseSpecialist("metrics-analyst", content)
	assert.NoError(t, err)
	assert.Equal(t, "metrics-analyst", rs.Name)
	assert.Equal(t, "Analyze metric anomalies for a service.", rs.Description)
	assert.Equal(t, "gpt-4o", rs.Model)
	assert.Equal(t, "query, dashboards", rs.Tools)
	// Body becomes the system prompt; frontmatter is stripped.
	assert.Contains(t, rs.SystemPrompt, "You are a metrics analysis expert.")
	assert.NotContains(t, rs.SystemPrompt, "description:")
}

func TestParseSpecialist_NoFrontmatter(t *testing.T) {
	rs, err := parseSpecialist("plain", "You are a plain specialist with no metadata.")
	assert.NoError(t, err)
	assert.Empty(t, rs.Description)
	assert.Empty(t, rs.Model)
	assert.Empty(t, rs.Tools)
	assert.Equal(t, "You are a plain specialist with no metadata.", rs.SystemPrompt)
}

func TestParseSpecialist_UnterminatedFrontmatterIsBody(t *testing.T) {
	// A leading "---" with no closing fence is treated as ordinary Markdown (whole body).
	rs, err := parseSpecialist("x", "---\ndescription: no closing fence\nstill body")
	assert.NoError(t, err)
	assert.Empty(t, rs.Description)
	assert.Contains(t, rs.SystemPrompt, "no closing fence")
}

func TestParseSpecialist_MalformedFrontmatterErrors(t *testing.T) {
	// Valid delimiters but non-YAML content between them → parse error, caller skips the row.
	content := "---\n" +
		"description: [unterminated\n" +
		"---\n" +
		"body"
	_, err := parseSpecialist("x", content)
	assert.Error(t, err)
}
