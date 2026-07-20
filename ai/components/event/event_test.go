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
	"testing"

	"github.com/stretchr/testify/assert"

	agentmodel "github.com/lindb/linsight/ai/model"
)

func TestConvContext_Seq(t *testing.T) {
	c := NewConvContext(0, "", 0, "")
	assert.Equal(t, 1, c.NextSeq())
	assert.Equal(t, 2, c.NextSeq())

	// SetSeq primes the counter so the next NextSeq continues from n+1 — the resume path
	// uses this to continue the interrupted run's sequence instead of restarting from 1.
	c.SetSeq(10)
	assert.Equal(t, 11, c.NextSeq())
	assert.Equal(t, 12, c.NextSeq())
}

func TestConvContext_Fork(t *testing.T) {
	c := NewConvContext(1, "conv-1", 7, "run-1")
	assert.Equal(t, 1, c.NextSeq())

	// Fork shares the seq counter but tags AgentID/ParentToolCallID.
	f := c.Fork("specialist-a", "call-1")
	assert.Equal(t, "specialist-a", f.AgentID)
	assert.Equal(t, "call-1", f.ParentToolCallID)
	assert.Equal(t, "run-1", f.RunID)
	// Shared counter: continues from the parent's value, not restarting from 1.
	assert.Equal(t, 2, f.NextSeq())
	assert.Equal(t, 3, c.NextSeq())
}

// TestStepEvent_OnFinishIdempotent verifies OnFinish is safe to call more than once — the
// normal graph path and the error-recovery finalizeError both close the step, and a
// double-close must not emit a second STEP_FINISHED frame or persist a duplicate row.
func TestStepEvent_OnFinishIdempotent(t *testing.T) {
	w := &fakeSSEWriter{}
	s := &fakeSaver{}
	e := NewStepEvent(w, s, NewConvContext(1, "conv-1", 7, "run-1"))

	e.OnStart(context.Background(), "step-1", "Deep Analysis")
	e.OnFinish(context.Background())
	e.OnFinish(context.Background()) // second close must be a no-op

	assert.Equal(t, 1, w.stepStarted)
	assert.Equal(t, 1, w.stepFinished, "double OnFinish must emit exactly one STEP_FINISHED")

	// Exactly two rows: one step_started, one step_finished (no duplicate).
	var started, finished int
	for _, row := range s.saved {
		switch row.Type {
		case agentmodel.EventTypeStepStarted:
			started++
		case agentmodel.EventTypeStepFinished:
			finished++
		}
	}
	assert.Equal(t, 1, started)
	assert.Equal(t, 1, finished, "double OnFinish must persist exactly one step_finished row")
}
