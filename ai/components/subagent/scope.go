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

// Package subagent provides a standalone, reusable ReAct runner that executes a
// sub-agent (a specialist handoff or a forked skill) in its own context while
// streaming the full process — reasoning, tool calls, and text — as AG-UI SSE
// events and persisting them to the DB, tagged with the sub-agent's AgentID and the
// host tool_call that spawned it (ParentToolCallID).
//
// The package deliberately does NOT import the agent package: the host constructs a
// Runner and injects it (plus a per-request Scope) into the request context. The
// skills and multiagent tools then resolve the Runner from context and delegate to
// it, so one ReAct implementation backs both specialist handoff and skill fork.
package subagent

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	agentapi "github.com/lindb/linsight/ai/api"
	aievent "github.com/lindb/linsight/ai/components/event"
)

// RunMode selects the initial-context strategy for a sub-agent run.
type RunMode int

const (
	// ModeHandoff is a specialist handoff: an isolated context seeded with the
	// specialist system prompt and the delegated goal.
	ModeHandoff RunMode = iota
	// ModeSkillFork is a forked skill: an isolated blank slate whose system prompt is
	// the skill body.
	ModeSkillFork
	// ModeSkillForkWithContext is a forked skill that additionally inherits the parent
	// conversation's persisted history before the skill body.
	ModeSkillForkWithContext
)

// RunRequest is the task-specific input a consumer (handoff tool / skill tool) builds.
type RunRequest struct {
	// AgentID is the specialist or skill name; used for event attribution and cycle detection.
	AgentID string
	// SystemPrompt is the sub-agent persona (specialist prompt or skill body).
	SystemPrompt string
	// Goal is the delegated task (handoff). Empty for skill forks (the skill body is the task).
	Goal string
	// ToolWhitelist restricts the inherited tool set by name; empty means inherit all.
	ToolWhitelist []string
	// Mode selects the initial-context strategy.
	Mode RunMode
}

// Runner executes a sub-agent ReAct loop and returns the final assistant text.
// The caller (handoff/skill tool) post-processes the text — e.g. parses Evidence.
type Runner interface {
	Run(ctx context.Context, req RunRequest) (finalText string, err error)
}

// Scope carries the per-request primitives owned by the host. It is injected once
// (via WithScope) so sub-agent runs can stream/persist events on the same connection
// and derive their tagged ConvContext/SSEWriter from the host's.
type Scope struct {
	// Writer is the host SSE writer; the runner derives a Scoped() sub-writer from it.
	Writer agentapi.SSEWriter
	// Saver persists events; the AgentID/ParentToolCallID tagging comes from the forked ConvCtx.
	Saver aievent.EventSaver
	// ConvCtx is the host run context; the runner Fork()s it (sharing the seq counter).
	ConvCtx *aievent.ConvContext
	// BaseTools is the full host-resolved org tool set the runner filters from.
	BaseTools []tool.BaseTool
	// History returns the parent conversation's persisted history for fork_with_context.
	// May be nil (fork_with_context then degrades to a blank slate).
	History func(ctx context.Context) ([]*schema.Message, error)
}

type ctxKey int8

const (
	ctxRunner ctxKey = iota
	ctxScope
	ctxParentToolCallID
	ctxAgentStack
)

// WithRunner stores the shared Runner in ctx so handoff/skill tools can resolve it.
func WithRunner(ctx context.Context, r Runner) context.Context {
	return context.WithValue(ctx, ctxRunner, r)
}

// RunnerFrom returns the Runner injected by the host, or (nil, false) when absent
// (e.g. unit tests invoking a tool directly), so callers can fall back gracefully.
func RunnerFrom(ctx context.Context) (Runner, bool) {
	r, ok := ctx.Value(ctxRunner).(Runner)
	return r, ok
}

// WithScope stores the per-request Scope in ctx.
func WithScope(ctx context.Context, s *Scope) context.Context {
	return context.WithValue(ctx, ctxScope, s)
}

// ScopeFrom returns the per-request Scope, or (nil, false) when absent.
func ScopeFrom(ctx context.Context) (*Scope, bool) {
	s, ok := ctx.Value(ctxScope).(*Scope)
	return s, ok
}

// WithParentToolCallID records the id of the tool_call currently being executed so a
// sub-agent tool (handoff/skill) invoked by it can tag its events with the parent id.
func WithParentToolCallID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxParentToolCallID, id)
}

// ParentToolCallIDFrom returns the enclosing tool_call id, or "" at the host top level.
func ParentToolCallIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxParentToolCallID).(string)
	return id
}

// agentStackFrom returns the names of sub-agents currently on the active call stack
// (outermost first). Used for cycle detection and depth limiting.
func agentStackFrom(ctx context.Context) []string {
	s, _ := ctx.Value(ctxAgentStack).([]string)
	return s
}

// pushAgent returns a ctx whose active-agent stack has name appended. A fresh slice is
// allocated each time so sibling branches never observe each other's pushes.
func pushAgent(ctx context.Context, name string) context.Context {
	old := agentStackFrom(ctx)
	next := make([]string, len(old)+1)
	copy(next, old)
	next[len(old)] = name
	return context.WithValue(ctx, ctxAgentStack, next)
}
