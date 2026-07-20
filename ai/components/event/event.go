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

// Package event encapsulates the SSE streaming + DB persistence lifecycle for each
// AG-UI event type. Each event object (AssistantTextEvent, ReasoningEvent,
// ToolCallEvent, RunEvent) owns its own Start→Content→End transition so callers
// only see typed method calls instead of raw SSEWriter + DB calls.
package event

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/cloudwego/eino/schema"

	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/uuid"
)

// ConvContext holds the immutable identifiers for one agent run and a per-run
// atomic sequence counter so all events in the run are globally ordered.
// It is created once per chatRun and shared (by pointer) across all event objects
// so the sequence is monotonically increasing across concurrent tool callbacks.
//
// seq is a *shared* pointer: Fork() derives a scoped ConvContext (for a sub-agent
// run) that carries a different AgentID/ParentToolCallID but SHARES the same seq
// counter, so host and all nested sub-agent events keep one run-global monotonic
// order for deterministic replay.
type ConvContext struct {
	ConvID  int64
	ConvUID string
	UserID  int64
	RunID   string
	// AgentID is the sub-agent (specialist/fork skill) name for events produced by a
	// forked sub-agent run; empty for host events.
	AgentID string
	// ParentToolCallID is the host tool_call id that spawned this sub-agent run; empty for host.
	ParentToolCallID string
	seq              *atomic.Int32
}

// NewConvContext creates a ConvContext with an initialised (host-scoped) sequence counter.
func NewConvContext(convID int64, convUID string, userID int64, runID string) *ConvContext {
	return &ConvContext{
		ConvID:  convID,
		ConvUID: convUID,
		UserID:  userID,
		RunID:   runID,
		seq:     new(atomic.Int32),
	}
}

// Fork derives a sub-agent-scoped ConvContext that tags every event it produces with
// agentID/parentToolCallID while SHARING the parent's seq counter (by pointer) so the
// whole run keeps a single monotonic sequence across host and nested sub-agents.
func (c *ConvContext) Fork(agentID, parentToolCallID string) *ConvContext {
	return &ConvContext{
		ConvID:           c.ConvID,
		ConvUID:          c.ConvUID,
		UserID:           c.UserID,
		RunID:            c.RunID,
		AgentID:          agentID,
		ParentToolCallID: parentToolCallID,
		seq:              c.seq,
	}
}

// NextSeq atomically increments and returns the next sequence number for this run.
func (c *ConvContext) NextSeq() int {
	return int(c.seq.Add(1))
}

// SetSeq resets the sequence counter to n so the next NextSeq() returns n+1.
// Used on a resume run that reuses the interrupted run's RunID: the counter is
// primed with that run's existing max sequence so resumed events continue the
// original run's monotonic ordering instead of colliding from 1.
func (c *ConvContext) SetSeq(n int) {
	c.seq.Store(int32(n))
}

// EventSaver abstracts DB persistence so the event package does not import the
// service layer directly, which would create a circular dependency.
type EventSaver interface {
	SaveEvent(ctx context.Context, evt *agentmodel.Event) error
}

// evtOption is a functional option applied to an agentmodel.Event after construction.
type evtOption func(*agentmodel.Event)

// withStatus sets the status column on the DB event row.
func withStatus(status agui.MessageStatus) evtOption {
	return func(e *agentmodel.Event) {
		e.Status = string(status)
	}
}

// withMessageUID overrides the auto-generated MessageUID so the persisted row carries
// the same AG-UI message ID that was emitted over SSE. This keeps live-stream and
// history-replay message IDs identical for events whose ID the frontend correlates
// (assistant text, reasoning).
func withMessageUID(uid string) evtOption {
	return func(e *agentmodel.Event) {
		if uid != "" {
			e.MessageUID = uid
		}
	}
}

// withToolCallID sets the tool_call_id column on the DB event row.
func withToolCallID(callID string) evtOption {
	return func(e *agentmodel.Event) {
		e.ToolCallID = callID
	}
}

// withToolName sets the tool_name column on the DB event row.
func withToolName(name string) evtOption {
	return func(e *agentmodel.Event) {
		e.ToolName = name
	}
}

// withExtra serialises extra and attaches it to the DB event row.
// A nil map is a no-op.
func withExtra(extra map[string]any) evtOption {
	return func(e *agentmodel.Event) {
		if extra != nil {
			e.Extra, _ = json.Marshal(extra)
		}
	}
}

// withCode sets the error code column for run_error events.
func withCode(code string) evtOption {
	return func(e *agentmodel.Event) {
		e.Code = code
	}
}

// withRawMessage serialises sm and stores it in RawMessage for LLM context reconstruction.
func withRawMessage(sm *schema.Message) evtOption {
	return func(e *agentmodel.Event) {
		if sm != nil {
			e.RawMessage, _ = json.Marshal(sm)
		}
	}
}

// newDBEvent builds an agentmodel.Event ready for DB insertion.
// stepID groups all events that belong to the same model-call iteration; pass ""
// for user_message and run_error events that are not part of any iteration.
func newDBEvent(
	convCtx *ConvContext,
	evtType string,
	stepID string,
	role schema.RoleType,
	content string,
	elapsedMs int64,
	opts ...evtOption,
) *agentmodel.Event {
	e := &agentmodel.Event{
		RunID:            convCtx.RunID,
		ConvUID:          convCtx.ConvUID,
		MessageUID:       uuid.GenerateShortUUID(),
		StepID:           stepID,
		Role:             role,
		Type:             evtType,
		Status:           string(agui.StatusComplete),
		Content:          content,
		Sequence:         convCtx.NextSeq(),
		ElapsedMs:        elapsedMs,
		AgentID:          convCtx.AgentID,
		ParentToolCallID: convCtx.ParentToolCallID,
	}
	e.CreatedBy = convCtx.UserID
	e.UpdatedBy = convCtx.UserID
	for _, opt := range opts {
		opt(e)
	}
	return e
}
