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

// AssistantTextEvent manages the TEXT_MESSAGE_* lifecycle and DB persistence for the
// assistant's visible text, segmented per model-call iteration (step).
//
// Design invariant: a single AssistantTextEvent instance is reused across the whole
// chatRun (it also acts as the factory for per-step ReasoningEvents), but each step is
// an independent text SEGMENT. BeginSegment opens a fresh segment with its own msgUID
// and timer; OnEnd closes the SSE frame AND persists that segment immediately (mirroring
// ReasoningEvent). This keeps live-stream and history-replay identical: one assistant_text
// row per step, each with a stable msgUID and per-step elapsed — instead of one accumulated
// row written at finalize with a run-cumulative elapsed.
type AssistantTextEvent struct {
	curMsgUID string          // current segment's AG-UI message ID (per step)
	stepID    string          // current segment's model-call iteration ID
	segStart  time.Time       // set on first OnContent of the current segment
	started   bool            // TEXT_MESSAGE_START has been sent for the current segment
	segSB     strings.Builder // accumulates the current segment's text for DB persistence

	writer  agentapi.SSEWriter
	saver   EventSaver
	convCtx *ConvContext
}

// NewAssistantTextEvent allocates an AssistantTextEvent for one chatRun.
// The per-step message ID is supplied later via BeginSegment.
func NewAssistantTextEvent(writer agentapi.SSEWriter, saver EventSaver, convCtx *ConvContext) *AssistantTextEvent {
	return &AssistantTextEvent{
		writer:  writer,
		saver:   saver,
		convCtx: convCtx,
	}
}

// BeginSegment opens a fresh text segment for one model-call iteration.
// msgUID is the pre-generated AG-UI message ID for this step's text; stepID ties the
// persisted row to the other events of the same iteration. Must be called before the
// step's OnContent tokens arrive. START is emitted lazily on the first OnContent.
// Both arguments must be non-empty: an empty msgUID would reintroduce the exact
// live-SSE/history-replay ID mismatch this type is designed to prevent.
func (e *AssistantTextEvent) BeginSegment(msgUID, stepID string) {
	if msgUID == "" || stepID == "" {
		assistantLog.Warn("BeginSegment called with empty msgUID or stepID — segment disabled",
			logger.String("msgUID", msgUID), logger.String("stepID", stepID))
		e.curMsgUID = ""
		e.stepID = ""
		return
	}
	e.curMsgUID = msgUID
	e.stepID = stepID
	e.started = false
	e.segStart = time.Time{}
	e.segSB.Reset()
}

// OnContent handles one text delta token from the LLM.
// Lazily opens TEXT_MESSAGE_START and records segStart on the first call of the segment.
// No-op when BeginSegment has not been called (curMsgUID is empty).
func (e *AssistantTextEvent) OnContent(delta string) {
	if delta == "" || e.curMsgUID == "" {
		return
	}
	if !e.started {
		e.segStart = time.Now()
		e.writer.WriteTextMessageStart(e.curMsgUID, schema.Assistant)
		e.started = true
	}
	e.segSB.WriteString(delta)
	e.writer.WriteTextMessageContent(e.curMsgUID, delta)
}

// OnEnd closes the TEXT_MESSAGE_END SSE frame for the current segment and persists the
// segment row to DB. Called at the end of each model invocation — every segment is
// persisted immediately (like ReasoningEvent). The persisted row carries curMsgUID so
// history replay emits the same message ID that was streamed live.
func (e *AssistantTextEvent) OnEnd(ctx context.Context) {
	if !e.started {
		return
	}
	var elapsedMs int64
	if !e.segStart.IsZero() {
		elapsedMs = time.Since(e.segStart).Milliseconds()
	}
	e.writer.WriteTextMessageEnd(e.curMsgUID, elapsedMs)
	e.started = false

	if e.segSB.Len() == 0 {
		return
	}

	sm := &schema.Message{
		Role:    schema.Assistant,
		Content: e.segSB.String(),
	}
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeAssistantText,
		e.stepID,
		schema.Assistant,
		e.segSB.String(),
		elapsedMs,
		withMessageUID(e.curMsgUID),
		withRawMessage(sm),
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		assistantLog.Warn("failed to persist assistant text event", logger.Error(err))
	}
}

// CurrentMsgUID returns the current segment's message ID, used as the parentMessageId
// for tool calls produced in the same step.
func (e *AssistantTextEvent) CurrentMsgUID() string {
	return e.curMsgUID
}

// IsStarted reports whether a TEXT_MESSAGE_START frame is currently open.
// Used by error-recovery defers to close (and persist) the segment before RUN_ERROR.
func (e *AssistantTextEvent) IsStarted() bool {
	return e.started
}

// NewReasoningEvent allocates a fresh ReasoningEvent for one model-call iteration.
// Each model call gets its own instance so reasoning from multiple ReAct rounds
// is stored as independent rows with independent timing.
// stepID should be chatRun.currentStepID, generated at the start of the iteration.
func (e *AssistantTextEvent) NewReasoningEvent(stepID string) *ReasoningEvent {
	return &ReasoningEvent{
		msgUID:  uuid.GenerateShortUUID(),
		stepID:  stepID,
		writer:  e.writer,
		saver:   e.saver,
		convCtx: e.convCtx,
	}
}
