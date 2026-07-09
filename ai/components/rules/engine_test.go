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

package rules

import (
	"context"
	"strings"
	"testing"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
)

func testCtx(orgID int64) context.Context {
	return context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
		Org:  &model.Org{BaseModel: model.BaseModel{ID: orgID}},
		User: &model.User{BaseModel: model.BaseModel{ID: 1}},
	})
}

func TestParseRuleContent(t *testing.T) {
	t.Run("full frontmatter", func(t *testing.T) {
		rr, err := parseRuleContent("---\n" +
			"trigger_tools: [Linsight_Datasource_Query]\n" +
			"trigger_keywords: [SQL, 数据源]\n" +
			"variables: {\"env\": \"prod\"}\n" +
			"---\n" +
			"### Constraints\nBody text here.")
		assert.NoError(t, err)
		// tools/keywords are lower-cased for case-insensitive matching.
		assert.Equal(t, []string{"linsight_datasource_query"}, rr.TriggerTools)
		assert.Equal(t, []string{"sql", "数据源"}, rr.TriggerKeywords)
		assert.JSONEq(t, `{"env":"prod"}`, rr.VariablesJSON)
		assert.Contains(t, rr.Body, "Body text here.")
		assert.NotContains(t, rr.Body, "trigger_tools")
	})

	t.Run("no frontmatter is always-on body", func(t *testing.T) {
		rr, err := parseRuleContent("Just a plain markdown rule.")
		assert.NoError(t, err)
		assert.Empty(t, rr.TriggerTools)
		assert.Empty(t, rr.TriggerKeywords)
		assert.Empty(t, rr.VariablesJSON)
		assert.Equal(t, "Just a plain markdown rule.", rr.Body)
	})

	t.Run("unterminated frontmatter is treated as plain body", func(t *testing.T) {
		// A leading "---" with no closing fence is a Markdown thematic break, not a
		// malformed frontmatter block: keep the whole content as an always-on rule.
		rr, err := parseRuleContent("---\ntrigger_tools: [x]\nno closing delimiter")
		assert.NoError(t, err)
		assert.Empty(t, rr.TriggerTools)
		assert.Contains(t, rr.Body, "no closing delimiter")
	})

	t.Run("malformed yaml in a closed fence errors", func(t *testing.T) {
		_, err := parseRuleContent("---\ntrigger_tools: [unclosed\n---\nbody")
		assert.Error(t, err)
	})

	t.Run("nested yaml variables re-emitted as json", func(t *testing.T) {
		rr, err := parseRuleContent("---\nvariables:\n  env: prod\n  regions:\n    - us\n    - eu\n---\nbody")
		assert.NoError(t, err)
		assert.JSONEq(t, `{"env":"prod","regions":["us","eu"]}`, rr.VariablesJSON)
	})
}

// buildEngine returns an engine with a manually-populated cache (no DB needed).
func buildEngine(global []*RuntimeRule, perOrg map[int64][]*RuntimeRule) *DBRuleEngine {
	e := NewDBRuleEngine(nil, 0)
	e.globalRules = global
	if perOrg != nil {
		e.orgRules = perOrg
	}
	return e
}

func inputWithText(text string) *einomodel.CallbackInput {
	return &einomodel.CallbackInput{Messages: []*schema.Message{{Role: schema.User, Content: text}}}
}

func inputWithToolCall(name string) *einomodel.CallbackInput {
	return &einomodel.CallbackInput{Messages: []*schema.Message{{
		Role:      schema.Assistant,
		ToolCalls: []schema.ToolCall{{Function: schema.FunctionCall{Name: name}}},
	}}}
}

func TestMatchAndCompile_ResidentRule(t *testing.T) {
	e := buildEngine([]*RuntimeRule{
		{RuleName: "always", Body: "resident body"},
	}, nil)
	out, err := e.MatchAndCompile(testCtx(1), inputWithText("anything"))
	assert.NoError(t, err)
	assert.Contains(t, out, "## Rule: always")
	assert.Contains(t, out, "resident body")
}

func TestMatchAndCompile_ToolTrigger(t *testing.T) {
	e := buildEngine([]*RuntimeRule{
		{RuleName: "sql-rule", Body: "sql body", TriggerTools: []string{"linsight_datasource_query"}},
	}, nil)

	// Tool not invoked → no match.
	out, err := e.MatchAndCompile(testCtx(1), inputWithText("hello"))
	assert.NoError(t, err)
	assert.Empty(t, out)

	// Tool invoked (name is upper-case in the message; matching is case-insensitive).
	out, err = e.MatchAndCompile(testCtx(1), inputWithToolCall("Linsight_Datasource_Query"))
	assert.NoError(t, err)
	assert.Contains(t, out, "sql-rule")
}

func TestMatchAndCompile_KeywordTrigger(t *testing.T) {
	e := buildEngine([]*RuntimeRule{
		{RuleName: "kw-rule", Body: "kw body", TriggerKeywords: []string{"数据源"}},
	}, nil)

	out, err := e.MatchAndCompile(testCtx(1), inputWithText("请列出所有数据源"))
	assert.NoError(t, err)
	assert.Contains(t, out, "kw-rule")

	out, err = e.MatchAndCompile(testCtx(1), inputWithText("unrelated request"))
	assert.NoError(t, err)
	assert.Empty(t, out)
}

func TestMatchAndCompile_OrgScoping(t *testing.T) {
	e := buildEngine(
		[]*RuntimeRule{{RuleName: "global", Body: "g"}},
		map[int64][]*RuntimeRule{
			1: {{RuleName: "org1", Body: "o1"}},
			2: {{RuleName: "org2", Body: "o2"}},
		},
	)
	out, err := e.MatchAndCompile(testCtx(1), inputWithText("x"))
	assert.NoError(t, err)
	assert.Contains(t, out, "global")
	assert.Contains(t, out, "org1")
	assert.NotContains(t, out, "org2")
}

func TestMatchAndCompile_PriorityOrderAndVariables(t *testing.T) {
	e := buildEngine([]*RuntimeRule{
		{RuleName: "second", Body: "b2", Priority: 10},
		{RuleName: "first", Body: "b1", Priority: 1, VariablesJSON: `{"env":"prod"}`},
	}, nil)
	out, err := e.MatchAndCompile(testCtx(1), inputWithText("x"))
	assert.NoError(t, err)
	// Lower priority first.
	assert.Less(t, strings.Index(out, "first"), strings.Index(out, "second"))
	// Variables line rendered verbatim for the rule that declares them.
	assert.Contains(t, out, "- Environment variables: {\"env\":\"prod\"}")
}

func TestMatchAndCompile_NilInputAndEmptyCache(t *testing.T) {
	e := buildEngine(nil, nil)
	out, err := e.MatchAndCompile(testCtx(1), nil)
	assert.NoError(t, err)
	assert.Empty(t, out)

	out, err = e.MatchAndCompile(testCtx(1), inputWithText("x"))
	assert.NoError(t, err)
	assert.Empty(t, out)
}

func TestValidateContent(t *testing.T) {
	assert.NoError(t, ValidateContent("plain body"))
	assert.NoError(t, ValidateContent("---\ntrigger_tools: [a]\n---\nbody"))
	// Unterminated leading fence is valid (plain-body fallback); malformed closed YAML is not.
	assert.NoError(t, ValidateContent("---\nunterminated"))
	assert.Error(t, ValidateContent("---\ntrigger_tools: [unclosed\n---\nbody"))
}
