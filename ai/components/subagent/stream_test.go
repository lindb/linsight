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

package subagent

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestBuildStepTitlePrompt_EmptyWhenNoContentOrTools(t *testing.T) {
	assert.Empty(t, BuildStepTitlePrompt("", nil))
	assert.Empty(t, BuildStepTitlePrompt("   \n\t ", []schema.ToolCall{}))
}

func TestBuildStepTitlePrompt_IncludesContent(t *testing.T) {
	p := BuildStepTitlePrompt("Here is the datasource list.", nil)
	assert.NotEmpty(t, p)
	assert.Contains(t, p, "Here is the datasource list.")
	assert.Contains(t, p, "at most 6 words")
}

func TestBuildStepTitlePrompt_IncludesToolCalls(t *testing.T) {
	tcs := []schema.ToolCall{
		{Function: schema.FunctionCall{Name: "linsight_datasource_query", Arguments: `{"sql":"select 1"}`}},
		{Function: schema.FunctionCall{Name: "linsight_list_datasources"}},
	}
	p := BuildStepTitlePrompt("", tcs)
	assert.Contains(t, p, "linsight_datasource_query")
	assert.Contains(t, p, "select 1")
	assert.Contains(t, p, "linsight_list_datasources")
}

func TestBuildStepTitlePrompt_SkipsUnnamedToolCalls(t *testing.T) {
	// A tool call with no name and no content must not produce a prompt.
	tcs := []schema.ToolCall{{Function: schema.FunctionCall{Name: ""}}}
	assert.Empty(t, BuildStepTitlePrompt("", tcs))
}

func TestSanitizeStepTitle(t *testing.T) {
	assert.Equal(t, "Query datasource", sanitizeStepTitle("  \"Query datasource\" \n"))
	assert.Equal(t, "Summarize logs", sanitizeStepTitle("'Summarize logs'"))
	// Over-long titles are capped at 60 runes.
	long := strings.Repeat("x", 100)
	assert.Len(t, []rune(sanitizeStepTitle(long)), 60)
}
