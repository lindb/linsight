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
	sseStepID   string      // per-iteration UUID for SSE STEP_STARTED/STEP_FINISHED pairing
	modelStepID string      // stored at OnStart; used by OnFinish to write step_finished DB row
	name        string      // display name: set to "Step N" at OnStart, may be overridden by SetName
	startTime   time.Time   // set in OnStart; used to compute elapsed in OnFinish
	finished    atomic.Bool // idempotent guard: OnFinish is a no-op after the first call

	// nameCh, when set via SetNameSource, delivers an asynchronously generated step title.
	// OnFinish reads it (once) before writing the final name, so title generation can run
	// concurrently with tool execution and be joined at step close.
	nameCh <-chan string

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
// tool tokens are emitted. Emits STEP_STARTED SSE and persists the step_started DB row
// immediately. Since this runs in the single model-callback goroutine before any
// concurrent tool callbacks are dispatched, the sequence number is guaranteed to be
// lower than all subsequent tool_call/tool_result rows.
func (e *StepEvent) OnStart(ctx context.Context, modelStepID string, name string) {
	e.startTime = time.Now()
	e.modelStepID = modelStepID
	e.name = name

	// SSE: open the step before reasoning begins so the frontend groups the full iteration.
	e.writer.WriteStepStarted(e.sseStepID, name)

	// DB: persist step_started immediately.
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepStarted,
		modelStepID,
		schema.Assistant,
		"",
		0,
		withToolName(name),
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_started event", logger.Error(err))
	}
}

// SetName overrides the step's display name after OnStart (e.g. with a model-generated
// title known only once streaming has begun). It is a no-op for the SSE STEP_STARTED
// frame already emitted; the new name takes effect on the STEP_FINISHED frame and the
// step_finished DB row. Passing an empty name leaves the existing name unchanged.
func (e *StepEvent) SetName(name string) {
	if name != "" {
		e.name = name
	}
}

// SetNameSource registers a channel that will deliver an asynchronously generated step
// title (e.g. produced concurrently with tool execution). OnFinish reads a single value
// from it before writing the final name; a non-empty value overrides the placeholder.
// The channel MUST be buffered (size >= 1) and receive exactly one value so OnFinish never
// blocks indefinitely and the producing goroutine never leaks if OnFinish is skipped.
func (e *StepEvent) SetNameSource(ch <-chan string) {
	e.nameCh = ch
}

// OnFinish fires when all tool calls in the iteration complete, or immediately after
// model streaming ends for text-only iterations (no tool calls). Idempotent: subsequent
// calls after the first are silently ignored, allowing both the normal path and the
// error-recovery defer in chatRun.run() to call it safely without a double-close.
func (e *StepEvent) OnFinish(ctx context.Context) {
	if !e.finished.CompareAndSwap(false, true) {
		return
	}

	// Join any asynchronously generated title (may block until the producer sends, bounded
	// by the producer's own timeout). Empty leaves the "Step N" placeholder in place.
	if e.nameCh != nil {
		if n := <-e.nameCh; n != "" {
			e.name = n
		}
	}

	var elapsed int64
	if !e.startTime.IsZero() {
		elapsed = time.Since(e.startTime).Milliseconds()
	}

	// SSE: close the step with total iteration elapsed duration and the (possibly
	// overridden) step name.
	e.writer.WriteStepFinished(e.sseStepID, e.name, elapsed)

	// DB: persist step_finished for all iterations, carrying the final step name so
	// history replay can render the model-generated title.
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeStepFinished,
		e.modelStepID,
		schema.Assistant,
		"",
		elapsed,
		withToolName(e.name),
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		stepLog.Warn("failed to persist step_finished event", logger.Error(err))
	}
}
