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
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/uuid"
)

var stepLog = logger.GetLogger("ai", "StepEvent")

// StepEvent manages the STEP_STARTED/STEP_FINISHED SSE lifecycle and DB persistence
// for one model-call iteration (one ReAct round).
//
// A single StepEvent is created at the very start of OnEndWithStreamOutput — before any
// reasoning or tool tokens arrive — so the frontend can group the full iteration
// (reasoning → tool call(s) → tool result(s) → final text) under one step header.
//
// Two distinct IDs are used:
//   - sseStepID: generated in NewStepEvent; used only for STEP_STARTED/STEP_FINISHED
//     SSE correlation so the frontend can open and close the step UI element.
//   - modelStepID (passed to OnStart/OnFinish): the model-call iteration UUID stored
//     in DB step_id column, which correlates all events from the same ReAct round.
type StepEvent struct {
	sseStepID string    // per-iteration UUID for SSE STEP_STARTED/STEP_FINISHED pairing
	startTime time.Time // set in OnStart; used to compute elapsed in OnFinish
	finished  atomic.Bool // idempotent guard: OnFinish is a no-op after the first call

	writer  agentapi.SSEWriter
	saver   EventSaver
	convCtx *ConvContext
}

// NewStepEvent allocates a StepEvent for one model-call iteration.
func NewStepEvent(writer agentapi.SSEWriter, saver EventSaver, convCtx *ConvContext) *StepEvent {
	return &StepEvent{
		sseStepID: uuid.GenerateShortUUID(),
		writer:    writer,
		saver:     saver,
		convCtx:   convCtx,
	}
}

// OnStart fires at the very start of a model-call iteration, before any reasoning or
// tool tokens are emitted. Emits STEP_STARTED SSE with an empty step name (tool name
// is unknown at iteration start) and persists the step_started DB row.
// modelStepID should be chatRun.currentStepID, generated just before calling OnStart.
func (e *StepEvent) OnStart(ctx context.Context, modelStepID string) {
	e.startTime = time.Now()

	// SSE: open the step before reasoning begins so the frontend groups the full iteration.
	// StepName is empty because the tool name is not known until TOOL_CALL_START arrives.
	e.writer.WriteStepStarted(e.sseStepID, "")

	// DB: step_started row; Content and ToolName are intentionally empty because the
	// tool name is not known at iteration start. The sequence number is lower than any
	// reasoning row generated in the same iteration, which guarantees correct replay order.
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepStarted,
		modelStepID,
		schema.Assistant,
		"",
		0,
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_started event", logger.Error(err))
	}
}

// OnFinish fires when all tool calls in the iteration complete, or immediately after
// model streaming ends for text-only iterations (no tool calls). Idempotent: subsequent
// calls after the first are silently ignored, allowing both the normal path and the
// error-recovery defer in chatRun.run() to call it safely without a double-close.
func (e *StepEvent) OnFinish(ctx context.Context, modelStepID string) {
	if !e.finished.CompareAndSwap(false, true) {
		return
	}

	elapsed := time.Since(e.startTime).Milliseconds()

	// SSE: close the step.
	e.writer.WriteStepFinished(e.sseStepID)

	// DB: step_finished row with total iteration elapsed duration.
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepFinished,
		modelStepID,
		schema.Assistant,
		"",
		elapsed,
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_finished event", logger.Error(err))
	}
}
