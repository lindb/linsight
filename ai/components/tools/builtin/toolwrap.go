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

package builtin

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	toolscomp "github.com/lindb/linsight/ai/components/tools"
)

// toolWithExtra wraps an InvokableTool and injects static Extra metadata into
// its ToolInfo. toolutils.InferTool has no WithExtra option, so we use a thin
// wrapper to attach icon/render_type without modifying eino internals.
type toolWithExtra struct {
	tool.InvokableTool
	extra map[string]any
}

// Info delegates to the inner tool and sets Extra before returning, so callers
// (including callback.go) can read display metadata from ToolInfo.Extra.
func (t *toolWithExtra) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info, err := t.InvokableTool.Info(ctx)
	if err == nil && info != nil {
		info.Extra = t.extra
	}
	return info, err
}

// InvokableRun intercepts the tool result to strip runtime Extra before Eino
// forwards the response to the LLM, storing Extra in the request-scoped cache.
//
// ReAct mode (callID ≠ ""): parses ToolCallResult, stores Extra in the context
// cache keyed by callID so OnEnd can attach it to the SSE/DB event, then returns
// only string(tcr.Data) — the LLM-visible payload.
//
// Deep mode (callID == ""): returns the full ToolCallResult JSON unchanged so
// createToolsNode can parse it directly (no cache lookup needed).
//
// Non-ToolCallResult output (MCP tools, plain text): passes through unchanged.
func (t *toolWithExtra) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	result, err := t.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		return result, err
	}

	tcr, ok := toolscomp.ParseToolCallResult(result)
	if !ok {
		// Not a ToolCallResult (e.g., MCP tool or plain-text) — pass through unchanged.
		return result, nil
	}

	callID := compose.GetToolCallID(ctx)
	if callID == "" {
		// Deep mode: return full JSON so createToolsNode can parse it.
		return result, nil
	}

	// ReAct mode: cache Extra and return only the LLM-visible Data.
	if len(tcr.Extra) > 0 {
		if cache, ok := toolscomp.ExtraCacheFromContext(ctx); ok {
			cache.Store(callID, tcr.Extra)
		}
	}
	return string(tcr.Data), nil
}

// withExtraInfo wraps inner with static Extra metadata. The returned tool
// satisfies both tool.BaseTool and tool.InvokableTool.
func withExtraInfo(inner tool.InvokableTool, extra map[string]any) tool.InvokableTool {
	return &toolWithExtra{InvokableTool: inner, extra: extra}
}
