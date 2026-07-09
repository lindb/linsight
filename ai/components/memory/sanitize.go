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
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"
)

var sanitizeLog = logger.GetLogger("agent", "history_sanitize")

// SanitizeMessages enforces the tool-call/tool-result pairing invariant required by
// OpenAI-compatible chat APIs: every assistant message that carries tool_calls MUST be
// followed by a tool message responding to each tool_call_id, and every tool message MUST
// reference a tool_call emitted by a preceding assistant message.
//
// History rebuilt from the DB can violate this invariant in either direction:
//   - A tool_call survives but its tool_result was dropped (truncated by MemorySize, filtered
//     out because it had error status, or never persisted) -> the assistant would be sent with
//     an unanswered tool_call, producing "insufficient tool messages following tool_calls".
//   - A tool_result survives but its parent tool_call was dropped -> an orphaned tool message.
//
// For an assistant message it keeps only the tool_calls whose result is present; if that
// leaves no tool_calls and no textual content, the whole message is dropped. Tool messages
// are kept only when their tool_call_id matches a surviving tool_call.
//
// pendingToolCalls carries the tool calls of an unresolved interrupt (a resume run). These are
// legitimately unanswered — the resume flow is about to execute them — so they must survive
// even though no tool_result exists yet. Because the DB persists them as one row per call and
// an optional separate assistant_text row (which can reorder them), reconstructing the tail
// from history is fragile. Instead, any history tool_call belonging to a pending call is
// dropped in place and a single authoritative assistant message carrying ALL pendingToolCalls
// is appended at the very end. This guarantees the resume graph sees `lastMsg = assistant with
// the pending tool_calls` regardless of how the rows were split, ordered, or truncated.
// pendingToolCalls is nil for ordinary (non-resume) runs, leaving behaviour unchanged.
//
// The input order is otherwise preserved. `convUID` is used only for logging.
func SanitizeMessages(convUID string, msgs []*schema.Message, pendingToolCalls []schema.ToolCall) []*schema.Message {
	// Pass 1: collect the tool_call_ids for which a tool result exists, and the ids that
	// belong to the pending interrupt (rebuilt authoritatively from the record at the tail).
	resultIDs := make(map[string]bool)
	for _, m := range msgs {
		if m != nil && m.Role == schema.Tool && m.ToolCallID != "" {
			resultIDs[m.ToolCallID] = true
		}
	}
	pendingIDs := make(map[string]bool, len(pendingToolCalls))
	for _, tc := range pendingToolCalls {
		if tc.ID != "" {
			pendingIDs[tc.ID] = true
		}
	}

	// Pass 2: rebuild the list, keeping only tool_calls that have a matching result and only
	// tool messages that answer a surviving tool_call.
	survivingCallIDs := make(map[string]bool)
	out := make([]*schema.Message, 0, len(msgs)+1)
	for _, m := range msgs {
		if m == nil {
			continue
		}
		switch {
		case m.Role == schema.Assistant && len(m.ToolCalls) > 0:
			kept := m.ToolCalls[:0:0]
			var dropped []string
			for _, tc := range m.ToolCalls {
				switch {
				case pendingIDs[tc.ID]:
					// Pending interrupt call: drop the (possibly split/reordered) history row
					// regardless of whether a stray result exists; the authoritative copy is
					// appended once at the tail below. Checked before resultIDs so a pending
					// call is never both kept inline and re-appended (which would duplicate the
					// tool_call_id and reintroduce a pairing violation).
				case tc.ID != "" && resultIDs[tc.ID]:
					kept = append(kept, tc)
					survivingCallIDs[tc.ID] = true
				default:
					dropped = append(dropped, tc.ID)
				}
			}
			if len(dropped) > 0 {
				sanitizeLog.Warn("history: dropping assistant tool_calls without matching tool results",
					logger.String("conv.uid", convUID),
					logger.Any("dropped_call_ids", dropped))
			}
			// If no answered tool_calls remain and there is no textual content, the message
			// carries nothing useful and would still be invalid; drop it entirely.
			if len(kept) == 0 && strings.TrimSpace(m.Content) == "" {
				continue
			}
			m.ToolCalls = kept
			out = append(out, m)
		case m.Role == schema.Tool:
			if m.ToolCallID == "" || !survivingCallIDs[m.ToolCallID] {
				sanitizeLog.Warn("history: dropping orphaned tool message",
					logger.String("conv.uid", convUID),
					logger.String("tool_call_id", m.ToolCallID))
				continue
			}
			out = append(out, m)
		default:
			out = append(out, m)
		}
	}

	// Append the pending interrupt's tool calls as a single trailing assistant message so the
	// resume graph's Case-B guard (lastMsg is assistant-with-tool_calls) always matches and the
	// tool results appended during resume immediately follow their tool_calls.
	if len(pendingToolCalls) > 0 {
		out = append(out, &schema.Message{Role: schema.Assistant, ToolCalls: pendingToolCalls})
		sanitizeLog.Info("history: appended pending interrupt tool_calls at tail",
			logger.String("conv.uid", convUID),
			logger.Int("pending_count", len(pendingToolCalls)))
	}
	return out
}

// SummarizeMessages renders a compact one-line description of the message list for debugging
// history-context problems (e.g. tool-call/result pairing). It is safe to log: only roles,
// tool names, tool_call_ids and content lengths are emitted, never message content.
func SummarizeMessages(msgs []*schema.Message) string {
	parts := make([]string, 0, len(msgs))
	for i, m := range msgs {
		if m == nil {
			parts = append(parts, fmt.Sprintf("%d:<nil>", i))
			continue
		}
		switch {
		case m.Role == schema.Assistant && len(m.ToolCalls) > 0:
			ids := make([]string, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				ids = append(ids, fmt.Sprintf("%s(%s)", tc.Function.Name, tc.ID))
			}
			parts = append(parts, fmt.Sprintf("%d:assistant tool_calls=[%s]", i, strings.Join(ids, ",")))
		case m.Role == schema.Tool:
			parts = append(parts, fmt.Sprintf("%d:tool call_id=%s name=%s len=%d",
				i, m.ToolCallID, m.ToolName, len(m.Content)))
		default:
			parts = append(parts, fmt.Sprintf("%d:%s len=%d", i, m.Role, len(m.Content)))
		}
	}
	return strings.Join(parts, " | ")
}
