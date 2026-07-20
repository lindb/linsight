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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	aievent "github.com/lindb/linsight/ai/components/event"
	toolscomp "github.com/lindb/linsight/ai/components/tools"
)

var subLog = logger.GetLogger("agent", "subagent")

// ToolNeedsApproval reports whether ToolInfo.Extra flags _gov_need_approval=true.
// Sub-agents run without a human-in-the-loop gate, so such tools are filtered out of
// their tool set (governance stays a host-only concern).
func ToolNeedsApproval(extra map[string]any) bool {
	if extra == nil {
		return false
	}
	v, ok := extra["_gov_need_approval"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// PublicExtra returns a shallow copy of extra with all _gov_* keys stripped, so
// governance metadata never leaks to SSE frames or DB rows. Returns nil if empty.
func PublicExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return extra
	}
	result := make(map[string]any, len(extra))
	for k, v := range extra {
		if !strings.HasPrefix(k, "_gov_") {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// CloneToolExtra returns a shallow copy of the named tool's ToolInfo.Extra, or nil if
// the tool is not found or has no Extra. Used to attach display metadata (icon,
// render_type) to tool events without coupling to specific tool implementations.
func CloneToolExtra(ctx context.Context, tools []tool.BaseTool, name string) map[string]any {
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil || info == nil || info.Name != name {
			continue
		}
		if len(info.Extra) == 0 {
			return nil
		}
		m := make(map[string]any, len(info.Extra))
		for k, v := range info.Extra {
			m[k] = v
		}
		return m
	}
	return nil
}

// ParseToolResult unwraps a ToolCallResult envelope (if present) and injects raw_data
// into extra for rich frontend rendering (e.g. DataQueryView table).
func ParseToolResult(result string) (llmContent string, extra map[string]any) {
	if tcr, ok := toolscomp.ParseToolCallResult(result); ok {
		llmContent = string(tcr.Data)
		extra = tcr.Extra
		if extra == nil {
			extra = make(map[string]any)
		}
		var rawData json.RawMessage
		if json.Unmarshal(tcr.Data, &rawData) == nil {
			extra["raw_data"] = rawData
		}
		return llmContent, extra
	}
	return result, nil
}

// DispatchToolCalls runs fn for each tool call, concurrently up to maxParallel workers
// (a value <= 1, or a single call, runs sequentially), and returns the results in the
// ORIGINAL order regardless of completion order. Each worker writes a distinct result
// slot and the final slice is assembled after all workers finish, so replay ordering is
// deterministic and there is no shared-slice race. Safe because tool calls within one
// step are independent by construction.
func DispatchToolCalls(toolCalls []schema.ToolCall, maxParallel int, fn func(i int) *schema.Message) []*schema.Message {
	results := make([]*schema.Message, len(toolCalls))
	if maxParallel < 1 {
		maxParallel = 1
	}
	if maxParallel == 1 || len(toolCalls) <= 1 {
		for i := range toolCalls {
			results[i] = fn(i)
		}
		return results
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxParallel)
	for i := range toolCalls {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = fn(idx)
		}(i)
	}
	wg.Wait()
	return results
}

// RunToolCall executes a single tool call, emitting the TOOL_CALL_* lifecycle events on
// writer and returning the resulting tool message. Before invoking the tool it records
// tc.ID as the parent-tool-call id in ctx, so a sub-agent tool (handoff/skill) invoked
// here can attribute its own events to this call.
//
// It is safe to call concurrently for distinct tool calls within one step: each
// invocation owns its ToolCallEvent, the SSE writer serializes its own frames, the event
// sequence counter is atomic, and each goroutine writes a distinct result slot. A tool
// error (or panic) is converted into a tool message (never a Go error) so one failing
// tool never aborts its siblings.
//
// skipOnStart mirrors the host resume path: when true, TOOL_CALL_START was already
// emitted by a prior interrupt, so only the result is emitted here.
func RunToolCall(
	ctx context.Context,
	tc *schema.ToolCall,
	toolMap map[string]tool.InvokableTool,
	tools []tool.BaseTool,
	writer agentapi.SSEWriter,
	saver aievent.EventSaver,
	convCtx *aievent.ConvContext,
	parentMsgUID, stepID string,
	skipOnStart bool,
) (msg *schema.Message) {
	toolEvt := aievent.NewToolCallEvent(
		tc.ID, tc.Function.Name,
		parentMsgUID, stepID,
		writer, saver, convCtx,
	)

	// Recover from a panicking tool. REQUIRED for the parallel path: a panic in a worker
	// goroutine cannot be caught by the caller's recover, so without this it crashes the
	// process. Degrade to a tool-error message + event, like a returned error.
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("tool %q panicked: %v", tc.Function.Name, r)
			subLog.Error("tool call panicked", logger.String("tool", tc.Function.Name), logger.Error(panicErr))
			toolEvt.OnError(ctx, panicErr)
			msg = &schema.Message{
				Role:       schema.Tool,
				Content:    panicErr.Error(),
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
			}
		}
	}()

	if !skipOnStart {
		staticExtra := CloneToolExtra(ctx, tools, tc.Function.Name)
		toolEvt.OnStart(ctx, tc.Function.Arguments, PublicExtra(staticExtra))
	}

	it, ok := toolMap[tc.Function.Name]
	if !ok {
		toolErr := fmt.Errorf("tool %q not found", tc.Function.Name)
		toolEvt.OnError(ctx, toolErr)
		return &schema.Message{
			Role:       schema.Tool,
			Content:    toolErr.Error(),
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
		}
	}

	// Record this call as the parent so a sub-agent tool spawned here can attribute events.
	callCtx := WithParentToolCallID(ctx, tc.ID)
	result, runErr := it.InvokableRun(callCtx, tc.Function.Arguments)
	if runErr != nil {
		toolEvt.OnError(ctx, runErr)
		return &schema.Message{
			Role:       schema.Tool,
			Content:    runErr.Error(),
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
		}
	}
	llmContent, extra := ParseToolResult(result)
	toolEvt.OnEnd(ctx, llmContent, extra)
	return &schema.Message{
		Role:       schema.Tool,
		Content:    llmContent,
		ToolCallID: tc.ID,
		ToolName:   tc.Function.Name,
	}
}
