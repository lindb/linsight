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

package memory

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func assistantWithToolCalls(content string, callIDs ...string) *schema.Message {
	tcs := make([]schema.ToolCall, 0, len(callIDs))
	for _, id := range callIDs {
		tcs = append(tcs, schema.ToolCall{ID: id, Type: "function",
			Function: schema.FunctionCall{Name: "query"}})
	}
	return &schema.Message{Role: schema.Assistant, Content: content, ToolCalls: tcs}
}

func toolResult(callID, content string) *schema.Message {
	return &schema.Message{Role: schema.Tool, ToolCallID: callID, ToolName: "query", Content: content}
}

func TestSanitizeMessages(t *testing.T) {
	t.Run("keeps a well-formed tool_call/result pair", func(t *testing.T) {
		in := []*schema.Message{
			{Role: schema.User, Content: "hi"},
			assistantWithToolCalls("", "call_1"),
			toolResult("call_1", "ok"),
			{Role: schema.Assistant, Content: "done"},
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 4)
		assert.Equal(t, "call_1", out[1].ToolCalls[0].ID)
	})

	t.Run("drops assistant tool_call whose result is missing", func(t *testing.T) {
		// This is the exact scenario producing the 400 "insufficient tool messages" error:
		// the tool_call survived history truncation but its tool_result did not.
		in := []*schema.Message{
			{Role: schema.User, Content: "hi"},
			assistantWithToolCalls("", "call_missing"),
			{Role: schema.User, Content: "next question"},
		}
		out := SanitizeMessages("c1", in, nil)
		// The content-less assistant with an unanswered tool_call is dropped entirely.
		assert.Len(t, out, 2)
		assert.Equal(t, schema.User, out[0].Role)
		assert.Equal(t, schema.User, out[1].Role)
	})

	t.Run("keeps assistant text but strips unanswered tool_call", func(t *testing.T) {
		in := []*schema.Message{
			assistantWithToolCalls("here is my analysis", "call_missing"),
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 1)
		assert.Empty(t, out[0].ToolCalls)
		assert.Equal(t, "here is my analysis", out[0].Content)
	})

	t.Run("keeps only answered tool_calls in a multi-call message", func(t *testing.T) {
		in := []*schema.Message{
			assistantWithToolCalls("", "call_1", "call_2"),
			toolResult("call_1", "ok"),
			// call_2 result missing
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 2)
		assert.Len(t, out[0].ToolCalls, 1)
		assert.Equal(t, "call_1", out[0].ToolCalls[0].ID)
		assert.Equal(t, "call_1", out[1].ToolCallID)
	})

	t.Run("drops orphaned tool message", func(t *testing.T) {
		in := []*schema.Message{
			{Role: schema.User, Content: "hi"},
			toolResult("call_orphan", "stale result"),
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 1)
		assert.Equal(t, schema.User, out[0].Role)
	})

	t.Run("handles empty and nil input", func(t *testing.T) {
		assert.Empty(t, SanitizeMessages("c1", nil, nil))
		assert.Empty(t, SanitizeMessages("c1", []*schema.Message{nil}, nil))
	})

	t.Run("resume: appends pending tool_calls at the tail as a single assistant message", func(t *testing.T) {
		// Regression: a resume run must end with an assistant-with-tool_calls message so the
		// deep graph's Case-B guard matches. The pending tool_calls have no result yet.
		pending := []schema.ToolCall{
			{ID: "call_pending", Type: "function", Function: schema.FunctionCall{Name: "query"}},
		}
		in := []*schema.Message{
			{Role: schema.User, Content: "hi"},
			assistantWithToolCalls("", "call_1"),
			toolResult("call_1", "ok"),
			{Role: schema.Assistant, Content: "some analysis"},
		}
		out := SanitizeMessages("c1", in, pending)
		last := out[len(out)-1]
		assert.Equal(t, schema.Assistant, last.Role)
		assert.Len(t, last.ToolCalls, 1)
		assert.Equal(t, "call_pending", last.ToolCalls[0].ID)
	})

	t.Run("resume: dedups pending tool_call rows from history and appends once at tail", func(t *testing.T) {
		// The interrupt persisted the tool_call as a history row. Without dedup it would appear
		// twice (once unanswered mid-history, once at the tail). The mid-history copy must be
		// dropped (it has no result) and exactly one authoritative copy kept at the end.
		pending := []schema.ToolCall{
			{ID: "call_pending", Type: "function", Function: schema.FunctionCall{Name: "query"}},
		}
		in := []*schema.Message{
			{Role: schema.User, Content: "hi"},
			assistantWithToolCalls("", "call_pending"), // history row for the pending call
		}
		out := SanitizeMessages("c1", in, pending)
		// user + single tail assistant-with-tool_calls; the mid-history duplicate is gone.
		assert.Len(t, out, 2)
		assert.Equal(t, schema.User, out[0].Role)
		assert.Equal(t, schema.Assistant, out[1].Role)
		assert.Len(t, out[1].ToolCalls, 1)
		assert.Equal(t, "call_pending", out[1].ToolCalls[0].ID)
	})

	t.Run("reorders tool_result interleaved after assistant text (interrupt/resume)", func(t *testing.T) {
		// The exact interrupt/resume ordering bug: processInterrupt persists the assistant_text
		// row immediately, while the tool_result row is written only after the resume is
		// approved. Ordered by insert id the history becomes
		// assistant(tool_calls) -> assistant(text) -> tool(result), which puts a message between
		// the tool_call and its answer and produces the 400. Sanitize must move the result to sit
		// right after its tool_call.
		in := []*schema.Message{
			{Role: schema.User, Content: "add alert"},
			assistantWithToolCalls("Sure, I'll create an alert for test44", "call_1"),
			{Role: schema.Assistant, Content: "some later text"},
			toolResult("call_1", "created"),
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 4)
		assert.Equal(t, schema.User, out[0].Role)
		// assistant-with-tool_calls, then its result immediately, then the interleaved text.
		assert.Len(t, out[1].ToolCalls, 1)
		assert.Equal(t, "call_1", out[1].ToolCalls[0].ID)
		assert.Equal(t, "Sure, I'll create an alert for test44", out[1].Content)
		assert.Equal(t, schema.Tool, out[2].Role)
		assert.Equal(t, "call_1", out[2].ToolCallID)
		assert.Equal(t, schema.Assistant, out[3].Role)
		assert.Equal(t, "some later text", out[3].Content)
	})

	t.Run("reorders multiple tool_results to declaration order next to their call", func(t *testing.T) {
		in := []*schema.Message{
			assistantWithToolCalls("", "call_a", "call_b"),
			{Role: schema.Assistant, Content: "interleaved"},
			toolResult("call_b", "b-result"), // results arrive out of declaration order
			toolResult("call_a", "a-result"),
		}
		out := SanitizeMessages("c1", in, nil)
		assert.Len(t, out, 4)
		assert.Len(t, out[0].ToolCalls, 2)
		// Results sit immediately after the call, ordered by ToolCalls declaration (a before b).
		assert.Equal(t, schema.Tool, out[1].Role)
		assert.Equal(t, "call_a", out[1].ToolCallID)
		assert.Equal(t, schema.Tool, out[2].Role)
		assert.Equal(t, "call_b", out[2].ToolCallID)
		assert.Equal(t, "interleaved", out[3].Content)
	})

	t.Run("resume: multiple pending tool_calls merged into one tail message", func(t *testing.T) {
		pending := []schema.ToolCall{
			{ID: "call_a", Type: "function", Function: schema.FunctionCall{Name: "query"}},
			{ID: "call_b", Type: "function", Function: schema.FunctionCall{Name: "list"}},
		}
		in := []*schema.Message{{Role: schema.User, Content: "hi"}}
		out := SanitizeMessages("c1", in, pending)
		last := out[len(out)-1]
		assert.Len(t, last.ToolCalls, 2)
		assert.Equal(t, "call_a", last.ToolCalls[0].ID)
		assert.Equal(t, "call_b", last.ToolCalls[1].ID)
	})
}

func TestSummarizeMessages(t *testing.T) {
	in := []*schema.Message{
		{Role: schema.User, Content: "hello"},
		assistantWithToolCalls("", "call_1"),
		toolResult("call_1", "result body"),
	}
	got := SummarizeMessages(in)
	assert.Contains(t, got, "0:user")
	assert.Contains(t, got, "tool_calls=[query(call_1)]")
	assert.Contains(t, got, "tool call_id=call_1")
}
