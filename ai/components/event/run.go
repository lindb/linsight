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
	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
)

// RunEvent emits the run-lifecycle SSE frames (RUN_STARTED, RUN_FINISHED, RUN_ERROR).
// No DB persistence — run lifecycle is implied by the conversation and its events.
type RunEvent struct {
	writer agentapi.SSEWriter
}

// NewRunEvent allocates a RunEvent bound to the given SSEWriter.
func NewRunEvent(writer agentapi.SSEWriter) *RunEvent {
	return &RunEvent{writer: writer}
}

// OnStart emits RUN_STARTED.
func (e *RunEvent) OnStart(runID, convUID string) {
	e.writer.WriteRunStarted(runID, convUID)
}

// OnEnd emits RUN_FINISHED.
func (e *RunEvent) OnEnd(runID, convUID, title string) {
	e.writer.WriteRunFinished(runID, convUID, title)
}

// OnError emits RUN_ERROR.
func (e *RunEvent) OnError(runID string, code agui.ErrorCode, message string) {
	e.writer.WriteRunError(runID, code, message)
}

// OnInterrupt emits a RUN_FINISHED frame with Outcome=interrupt so the frontend can
// render the human-approval card(s) and echo back the interruptId to resume.
func (e *RunEvent) OnInterrupt(runID, convUID string, payload *agui.InterruptPayload) {
	e.writer.WriteRunInterrupt(runID, convUID, payload)
}
