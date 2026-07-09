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
	"github.com/cloudwego/eino/schema"
)

// toolWithExtra wraps an InvokableTool and injects static Extra metadata into
// its ToolInfo. toolutils.InferTool has no WithExtra option, so we use a thin
// wrapper to attach icon/render_type without modifying eino internals.
type toolWithExtra struct {
	tool.InvokableTool
	extra map[string]any
}

// Info delegates to the inner tool and sets Extra before returning, so callers
// can read display metadata from ToolInfo.Extra.
func (t *toolWithExtra) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info, err := t.InvokableTool.Info(ctx)
	if err == nil && info != nil {
		info.Extra = t.extra
	}
	return info, err
}

// InvokableRun runs the inner tool and returns its result unchanged. Builtin tools
// return a ToolCallResult JSON envelope; the graph's tools node parses it directly
// via toolscomp.ParseToolCallResult to separate the LLM-visible Data from runtime
// Extra. Non-ToolCallResult output (MCP tools, plain text) also passes through as-is.
func (t *toolWithExtra) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
}

// withExtraInfo wraps inner with static Extra metadata. The returned tool
// satisfies both tool.BaseTool and tool.InvokableTool.
func withExtraInfo(inner tool.InvokableTool, extra map[string]any) tool.InvokableTool {
	return &toolWithExtra{InvokableTool: inner, extra: extra}
}
