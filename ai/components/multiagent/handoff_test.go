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

package multiagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// stubGenerator is a hand-written specialistGenerator that captures the messages it
// receives and returns a canned reply, so tests can assert on context isolation without
// touching a real model.
type stubGenerator struct {
	gotMessages []*schema.Message
	reply       string
	err         error
}

func (s *stubGenerator) Generate(_ context.Context, input []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	s.gotMessages = input
	if s.err != nil {
		return nil, s.err
	}
	return &schema.Message{Role: schema.Assistant, Content: s.reply}, nil
}

func newSpec() *runtimeSpecialist {
	return &runtimeSpecialist{
		Name:         "metrics-analyst",
		Description:  "Analyze metric anomalies for a given service and time window.",
		SystemPrompt: "You are a metrics analysis expert.",
	}
}

func TestHandoffToolInfo(t *testing.T) {
	tool := newHandoffTool(newSpec(), &stubGenerator{})
	info, err := tool.Info(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "handoff_to_metrics-analyst", info.Name)
	assert.Equal(t, "Analyze metric anomalies for a given service and time window.", info.Desc)

	// The tool must declare a required "goal" parameter.
	js, err := info.ParamsOneOf.ToJSONSchema()
	assert.NoError(t, err)
	assert.Contains(t, js.Required, "goal")
	_, goalDefined := js.Properties.Get("goal")
	assert.True(t, goalDefined)
}

func TestHandoffToolInfoFallbackDesc(t *testing.T) {
	spec := newSpec()
	spec.Description = ""
	tool := newHandoffTool(spec, &stubGenerator{})
	info, err := tool.Info(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, info.Desc, "metrics-analyst")
}

func TestHandoffRunContextIsolation(t *testing.T) {
	stub := &stubGenerator{reply: `{"success":true,"summary":"cpu saturated on node-3"}`}
	tool := newHandoffTool(newSpec(), stub)

	out, err := tool.InvokableRun(context.Background(), `{"goal":"Investigate CPU on service payment"}`)
	assert.NoError(t, err)

	// Context isolation: exactly the specialist system prompt + the goal, no history.
	assert.Len(t, stub.gotMessages, 2)
	assert.Equal(t, schema.System, stub.gotMessages[0].Role)
	assert.Contains(t, stub.gotMessages[0].Content, "You are a metrics analysis expert.")
	assert.Contains(t, stub.gotMessages[0].Content, "Output Contract") // evidenceContract appended
	assert.Equal(t, schema.User, stub.gotMessages[1].Role)
	assert.Equal(t, "Investigate CPU on service payment", stub.gotMessages[1].Content)

	// Evidence handback: the tool result is the parsed evidence as JSON.
	var ev Evidence
	assert.NoError(t, json.Unmarshal([]byte(out), &ev))
	assert.True(t, ev.Success)
	assert.Equal(t, "cpu saturated on node-3", ev.Summary)
	assert.Equal(t, "metrics-analyst", ev.AgentID)
}

func TestHandoffRunEmptyGoal(t *testing.T) {
	stub := &stubGenerator{}
	tool := newHandoffTool(newSpec(), stub)

	out, err := tool.InvokableRun(context.Background(), `{"goal":"   "}`)
	assert.NoError(t, err) // tool never errors; failure is encoded in the evidence
	assert.Nil(t, stub.gotMessages, "model must not be called when goal is empty")

	var ev Evidence
	assert.NoError(t, json.Unmarshal([]byte(out), &ev))
	assert.False(t, ev.Success)
}

func TestHandoffRunGeneratorError(t *testing.T) {
	stub := &stubGenerator{err: errors.New("model unavailable")}
	tool := newHandoffTool(newSpec(), stub)

	out, err := tool.InvokableRun(context.Background(), `{"goal":"do something"}`)
	assert.NoError(t, err) // error is surfaced as unsuccessful evidence, not a Go error

	var ev Evidence
	assert.NoError(t, json.Unmarshal([]byte(out), &ev))
	assert.False(t, ev.Success)
	assert.Contains(t, ev.Error, "model unavailable")
}

func TestHandoffRunInvalidArgs(t *testing.T) {
	tool := newHandoffTool(newSpec(), &stubGenerator{})
	out, err := tool.InvokableRun(context.Background(), `not json`)
	assert.NoError(t, err)

	var ev Evidence
	assert.NoError(t, json.Unmarshal([]byte(out), &ev))
	assert.False(t, ev.Success)
}

func TestGetToolsForOrgVisibility(t *testing.T) {
	// Build a backend cache directly (no DB): one global specialist and two org-scoped.
	b := &DBBackend{
		proxy: &stubGenerator{},
		globalMap: map[string]*runtimeSpecialist{
			"global-analyst": {Name: "global-analyst"},
		},
		orgMaps: map[int64]map[string]*runtimeSpecialist{
			7: {
				"org-analyst":    {Name: "org-analyst"},
				"global-analyst": {Name: "org-analyst-shadow"}, // name collision: global wins
			},
			9: {"other-org": {Name: "other-org"}},
		},
	}

	tools, err := b.GetToolsForOrg(context.Background(), 7)
	assert.NoError(t, err)

	names := map[string]bool{}
	for _, tl := range tools {
		info, ierr := tl.Info(context.Background())
		assert.NoError(t, ierr)
		names[info.Name] = true
	}
	// Org 7 sees its own specialist plus the global one; the org's colliding name is
	// overridden by the global specialist (deduped by name), and org 9 is invisible.
	assert.True(t, names["handoff_to_org-analyst"])
	assert.True(t, names["handoff_to_global-analyst"])
	assert.False(t, names["handoff_to_other-org"])
	assert.Len(t, tools, 2)
}
