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
//
// DB persistence is conditional: step_started and step_finished rows are only written
// for iterations that contain at least one tool call. Reasoning-only iterations still
// emit SSE frames (so the frontend can render the step container) but produce no DB
// rows because there is nothing meaningful to replay. Flush() must be called from the
// first tool callback to commit the step_started row before any tool events are saved.
type StepEvent struct {
	sseStepID   string      // per-iteration UUID for SSE STEP_STARTED/STEP_FINISHED pairing
	modelStepID string      // stored at OnStart; used by Flush to write step_started DB row
	name        string      // stored at OnStart; used by Flush to write step_started DB row
	preAllocSeq int         // sequence number claimed in OnStart, before any tool callbacks fire
	startTime   time.Time   // set in OnStart; used to compute elapsed in OnFinish
	finished    atomic.Bool // idempotent guard: OnFinish is a no-op after the first call
	flushed     atomic.Bool // true once Flush() has written step_started to DB

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
// tool tokens are emitted. Emits STEP_STARTED SSE immediately so the frontend can open
// the step container, and stores modelStepID/name for later use by Flush.
//
// DB persistence is intentionally deferred: step_started is NOT written here because
// we do not yet know whether this iteration will produce tool calls. Call Flush() from
// the first tool callback to commit the DB row only when tools are confirmed.
func (e *StepEvent) OnStart(_ context.Context, modelStepID string, name string) {
	e.startTime = time.Now()
	e.modelStepID = modelStepID
	e.name = name
	// Pre-allocate the sequence number here, while still in the single model-callback
	// goroutine and before any concurrent tool callbacks can call NextSeq(). This
	// guarantees step_started.Sequence < all subsequent tool_call/tool_result sequences
	// even when Eino invokes tool callbacks concurrently.
	e.preAllocSeq = e.convCtx.NextSeq()

	// SSE: open the step before reasoning begins so the frontend groups the full iteration.
	e.writer.WriteStepStarted(e.sseStepID, name)
}

// Flush writes the step_started DB row exactly once. It must be called from the first
// tool callback (before any tool events are saved) to ensure the step_started row
// precedes all tool_call rows in sequence order.
// Idempotent: subsequent calls are silently ignored.
func (e *StepEvent) Flush(ctx context.Context) {
	if !e.flushed.CompareAndSwap(false, true) {
		return
	}
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepStarted,
		e.modelStepID,
		schema.Assistant,
		"",
		0,
		withToolName(e.name),
	)
	// Override with the pre-allocated sequence so step_started.Sequence is guaranteed
	// to be lower than any subsequent tool_call rows. newDBEvent calls NextSeq() internally
	// (wasting one counter slot), but correctness requires we use the value claimed in
	// OnStart, before concurrent tool callbacks could claim their own sequence numbers.
	evt.Sequence = e.preAllocSeq
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_started event", logger.Error(err))
	}
}

// OnFinish fires when all tool calls in the iteration complete, or immediately after
// model streaming ends for text-only iterations (no tool calls). Idempotent: subsequent
// calls after the first are silently ignored, allowing both the normal path and the
// error-recovery defer in chatRun.run() to call it safely without a double-close.
func (e *StepEvent) OnFinish(ctx context.Context) {
	if !e.finished.CompareAndSwap(false, true) {
		return
	}

	var elapsed int64
	if !e.startTime.IsZero() {
		elapsed = time.Since(e.startTime).Milliseconds()
	}

	// SSE: close the step with total iteration elapsed duration (always emitted).
	e.writer.WriteStepFinished(e.sseStepID, elapsed)

	// DB: only persist step_finished when Flush was called (i.e. the iteration had
	// tool calls). Reasoning-only iterations emit the SSE frame above so the frontend
	// can close the step container, but produce no DB rows because there is nothing
	// meaningful to replay in history.
	if !e.flushed.Load() {
		return
	}
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepFinished,
		e.modelStepID,
		schema.Assistant,
		"",
		elapsed,
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_finished event", logger.Error(err))
	}
}
