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
)

var reasoningLog = logger.GetLogger("ai", "ReasoningEvent")

// ReasoningEvent manages the REASONING_MESSAGE_* lifecycle and DB persistence for one
// model-call iteration's thinking/reasoning block.
//
// A new ReasoningEvent is created per model call (via AssistantTextEvent.NewReasoningEvent)
// so each ReAct iteration's reasoning is stored as an independent DB row with its own
// timing. The stepID ties it to the other events produced in the same model-call iteration.
type ReasoningEvent struct {
	msgUID    string
	stepID    string          // model-call iteration ID; stored in DB step_id column
	startTime time.Time       // set on first OnContent call
	started   bool            // REASONING_MESSAGE_START has been sent
	rsb       strings.Builder // accumulated reasoning text for DB persistence

	writer  agentapi.SSEWriter
	saver   EventSaver
	convCtx *ConvContext
}

// OnContent handles one reasoning delta token from the LLM.
// Lazily opens REASONING_MESSAGE_START on the first call and records startTime.
func (e *ReasoningEvent) OnContent(delta string) {
	if delta == "" {
		return
	}
	if !e.started {
		e.startTime = time.Now()
		e.writer.WriteReasoningMessageStart(e.msgUID)
		e.started = true
	}
	e.rsb.WriteString(delta)
	e.writer.WriteReasoningMessageContent(e.msgUID, delta)
}

// OnEnd closes the REASONING_MESSAGE_END SSE frame and persists the reasoning row to DB.
// Called at the end of each model invocation — every reasoning instance is persisted
// immediately (unlike AssistantTextEvent which defers persistence to Persist()).
func (e *ReasoningEvent) OnEnd(ctx context.Context) {
	if !e.started {
		return
	}
	e.writer.WriteReasoningMessageEnd(e.msgUID)
	e.started = false

	if e.rsb.Len() == 0 {
		return
	}

	elapsedMs := time.Since(e.startTime).Milliseconds()
	// Reasoning content is excluded from LLM context: models reject their own
	// reasoning tokens. RawMessage is intentionally omitted (no schema.Message).
	evt := newDBEvent(e.convCtx,
		agentmodel.EventTypeReasoning,
		e.stepID,
		schema.Assistant,
		e.rsb.String(),
		elapsedMs,
	)
	if err := e.saver.SaveEvent(ctx, evt); err != nil {
		reasoningLog.Warn("failed to persist reasoning event", logger.Error(err))
	}
}
