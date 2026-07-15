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
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	toolscomp "github.com/lindb/linsight/ai/components/tools"
)

// MemoryService is the minimal surface that memory tools require from the service layer.
// Defined locally to avoid an import cycle with ai/service.
type MemoryService interface {
	SaveMemoryFromTool(ctx context.Context, id, content, category string) (string, error)
	DeleteMemoryFromTool(ctx context.Context, uid string) (string, error)
}

// MemoryToolsFactory provides the save_memory and delete_memory Eino tools.
// Unlike BuiltinToolsFactory the tool set is the same for all orgs, so the
// tools slice is built once at construction time. Org scoping is enforced inside
// the service methods via util.GetUser(ctx).
type MemoryToolsFactory struct {
	tools []tool.BaseTool
}

// NewMemoryToolsFactory constructs the two memory management tools.
// Returns an error only if Eino fails to infer a tool's JSON schema.
func NewMemoryToolsFactory(memSrv MemoryService) (*MemoryToolsFactory, error) {
	saveTool, err := buildSaveMemoryTool(memSrv)
	if err != nil {
		return nil, err
	}
	deleteTool, err := buildDeleteMemoryTool(memSrv)
	if err != nil {
		return nil, err
	}
	return &MemoryToolsFactory{
		tools: []tool.BaseTool{saveTool, deleteTool},
	}, nil
}

// GetToolsForOrg returns all memory tools; they are org-agnostic at the tool layer
// (org scoping is enforced inside the service methods).
func (f *MemoryToolsFactory) GetToolsForOrg(_ context.Context, _ int64) ([]tool.BaseTool, error) {
	return f.tools, nil
}

// ---------------------------------------------------------------------------
// save_memory tool
// ---------------------------------------------------------------------------

// saveMemoryInput is the input schema for the save_memory tool.
// id empty = create a new memory; id non-empty = update the existing memory with that UID.
type saveMemoryInput struct {
	// ID is the memory UID to update. Leave empty to create a new memory entry.
	ID string `json:"id" jsonschema:"description=Memory UID to update. Leave empty to create a new entry."`
	// Content is the full memory text to persist.
	Content string `json:"content" jsonschema:"required,description=The memory content to persist. For updates replaces the entire previous content."`
	// Category is an optional tag for grouping related memories.
	Category string `json:"category,omitempty" jsonschema:"description=Optional category tag (e.g. preference\\, project\\, personal)."`
}

func buildSaveMemoryTool(memSrv MemoryService) (tool.BaseTool, error) {
	t, err := toolutils.InferTool(
		"save_memory",
		"Create or update a long-term memory entry for the current user. "+
			"Use this when the user explicitly asks to remember something, update existing information, "+
			"or change a previously stored fact. "+
			"Leave the id field empty to create a new memory; provide an id to update an existing one. "+
			"Requires human approval before the change is persisted.",
		func(ctx context.Context, input saveMemoryInput) (*toolscomp.ToolCallResult, error) {
			diff, err := memSrv.SaveMemoryFromTool(ctx, input.ID, input.Content, input.Category)
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(diff)
			if err != nil {
				return nil, err
			}
			return &toolscomp.ToolCallResult{Data: data}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return withExtraInfo(t, map[string]any{
		"icon":        "memory",
		"render_type": "MemoryEditCard",
		// Approval gate: the interruptCheck node will pause for human sign-off.
		"_gov_need_approval": true,
		// Frontend card hint for the approval UI.
		"_gov_card_type": "memory_edit",
	}), nil
}

// ---------------------------------------------------------------------------
// delete_memory tool
// ---------------------------------------------------------------------------

// deleteMemoryInput is the input schema for the delete_memory tool.
type deleteMemoryInput struct {
	// ID is the memory UID to delete.
	ID string `json:"id" jsonschema:"required,description=The UID of the memory entry to delete (from the user long-term memory list in the system prompt)."`
}

func buildDeleteMemoryTool(memSrv MemoryService) (tool.BaseTool, error) {
	t, err := toolutils.InferTool(
		"delete_memory",
		"Delete a long-term memory entry. "+
			"Use this when the user explicitly asks to forget or remove something they previously remembered. "+
			"The id must match one of the UIDs shown in the user long-term memory list. "+
			"Requires human approval before the deletion is persisted.",
		func(ctx context.Context, input deleteMemoryInput) (*toolscomp.ToolCallResult, error) {
			diff, err := memSrv.DeleteMemoryFromTool(ctx, input.ID)
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(diff)
			if err != nil {
				return nil, err
			}
			return &toolscomp.ToolCallResult{Data: data}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return withExtraInfo(t, map[string]any{
		"icon":        "memory",
		"render_type": "MemoryEditCard",
		// Approval gate: the interruptCheck node will pause for human sign-off.
		"_gov_need_approval": true,
		// Frontend card hint for the approval UI.
		"_gov_card_type": "memory_edit",
	}), nil
}

// ---------------------------------------------------------------------------
// toolWithExtra wrapper (local copy of builtin.toolWithExtra to avoid cross-package coupling)
// ---------------------------------------------------------------------------

// toolWithExtra wraps an InvokableTool and injects static Extra metadata into
// its ToolInfo. toolutils.InferTool has no WithExtra option, so we use a thin
// wrapper to attach icon/render_type/_gov_* metadata without modifying eino internals.
type toolWithExtra struct {
	tool.InvokableTool
	extra map[string]any
}

// Info delegates to the inner tool and sets Extra before returning.
func (t *toolWithExtra) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info, err := t.InvokableTool.Info(ctx)
	if err == nil && info != nil {
		info.Extra = t.extra
	}
	return info, err
}

// InvokableRun runs the inner tool and returns its result unchanged.
func (t *toolWithExtra) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
}

// withExtraInfo wraps inner with static Extra metadata.
func withExtraInfo(inner tool.InvokableTool, extra map[string]any) tool.InvokableTool {
	return &toolWithExtra{InvokableTool: inner, extra: extra}
}
