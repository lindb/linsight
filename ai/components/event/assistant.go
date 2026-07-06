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

package event

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/uuid"
)

var assistantLog = logger.GetLogger("ai", "AssistantTextEvent")

// AssistantTextEvent manages the TEXT_MESSAGE_* lifecycle and final DB persistence for
// one chatRun's accumulated text response.
//
// Design invariant: a single AssistantTextEvent instance spans the entire chatRun,
// accumulating text from ALL model invocations (ReAct can trigger multiple model calls).
// OnEnd closes the SSE frame after each model call but does NOT persist — Persist is
// called exactly once from finalize() or an error defer. This avoids the
// "persisted=true on first OnEnd → second model call text lost" bug.
type AssistantTextEvent struct {
	msgUID    string
	startTime time.Time       // set on very first OnContent call across all model invocations
	started   bool            // TEXT_MESSAGE_START has been sent for the current model call
	sb        strings.Builder // accumulates ALL text across ALL model invocations
	persisted bool            // idempotent guard; Persist() is a no-op if already called

	writer  agentapi.SSEWriter
	saver   EventSaver
	convCtx *ConvContext
}

// NewAssistantTextEvent allocates an AssistantTextEvent for one chatRun.
// msgUID is the pre-generated AG-UI message ID that the frontend uses to correlate
// TEXT_MESSAGE_* events with the persisted row.
func NewAssistantTextEvent(msgUID string, writer agentapi.SSEWriter, saver EventSaver, convCtx *ConvContext) *AssistantTextEvent {
	return &AssistantTextEvent{
		msgUID:  msgUID,
		writer:  writer,
		saver:   saver,
		convCtx: convCtx,
	}
}

// OnContent handles one text delta token from the LLM.
// Lazily opens TEXT_MESSAGE_START on the first call; records startTime on first-ever call.
func (e *AssistantTextEvent) OnContent(delta string) {
	if delta == "" {
		return
	}
	if e.sb.Len() == 0 {
		// First text token across all model calls in this chatRun.
		e.startTime = time.Now()
	}
	if !e.started {
		e.writer.WriteTextMessageStart(e.msgUID, schema.Assistant)
		e.started = true
	}
	e.sb.WriteString(delta)
	e.writer.WriteTextMessageContent(e.msgUID, delta)
}

// OnEnd closes the SSE TEXT_MESSAGE_END frame for the current model call.
// Resets started so the next model call can re-open the frame if needed.
// Does NOT write to DB — call Persist once from finalize() or an error defer.
func (e *AssistantTextEvent) OnEnd(_ context.Context) {
	if e.started {
		e.writer.WriteTextMessageEnd(e.msgUID)
		e.started = false
	}
}

// Persist writes the accumulated text as a single assistant_text event row.
// stepID should be rt.currentStepID (the last model-call iteration ID in the chatRun),
// which ties the text row to the final ReAct iteration.
// Idempotent: subsequent calls after the first are silently ignored.
func (e *AssistantTextEvent) Persist(ctx context.Context, stepID string) {
	if e.persisted {
		return
	}
	if e.sb.Len() == 0 {
		return
	}
	e.persisted = true

	var elapsedMs int64
	if !e.startTime.IsZero() {
		elapsedMs = time.Since(e.startTime).Milliseconds()
	}

	sm := &schema.Message{
		Role:    schema.Assistant,
		Content: e.sb.String(),
	}
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeAssistantText,
		stepID,
		schema.Assistant,
		e.sb.String(),
		elapsedMs,
		withRawMessage(sm),
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		assistantLog.Warn("failed to persist assistant text event", logger.Error(err))
	}
}

// HasContent reports whether any text has been accumulated.
// Used by callers to decide whether to call Persist before returning.
func (e *AssistantTextEvent) HasContent() bool {
	return e.sb.Len() > 0
}

// IsStarted reports whether a TEXT_MESSAGE_START frame is currently open.
// Used by error-recovery defers to close the frame before emitting RUN_ERROR.
func (e *AssistantTextEvent) IsStarted() bool {
	return e.started
}

// NewReasoningEvent allocates a fresh ReasoningEvent for one model-call iteration.
// Each model call gets its own instance so reasoning from multiple ReAct rounds
// is stored as independent rows with independent timing.
// stepID should be chatRun.currentStepID, generated at the start of OnEndWithStreamOutput.
func (e *AssistantTextEvent) NewReasoningEvent(stepID string) *ReasoningEvent {
	return &ReasoningEvent{
		msgUID:  uuid.GenerateShortUUID(),
		stepID:  stepID,
		writer:  e.writer,
		saver:   e.saver,
		convCtx: e.convCtx,
	}
}
