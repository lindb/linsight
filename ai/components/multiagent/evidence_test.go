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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // empty means nil expected
	}{
		{"plain object", `{"a":1}`, `{"a":1}`},
		{"leading prose", `Sure, here you go: {"a":1} done.`, `{"a":1}`},
		{"json code fence", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"bare code fence", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"nested braces", `{"a":{"b":2},"c":3}`, `{"a":{"b":2},"c":3}`},
		{"brace inside string", `{"a":"}{","b":1}`, `{"a":"}{","b":1}`},
		{"escaped quote in string", `{"a":"say \"hi\"","b":1}`, `{"a":"say \"hi\"","b":1}`},
		{"no object", `just text, no json here`, ""},
		{"unbalanced", `{"a":1`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSON(tc.in)
			if tc.want == "" {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestParseEvidence(t *testing.T) {
	t.Run("well-formed json", func(t *testing.T) {
		ev := parseEvidence("metrics-analyst", `{"agentId":"metrics-analyst","success":true,"summary":"cpu spiked"}`)
		assert.True(t, ev.Success)
		assert.Equal(t, "metrics-analyst", ev.AgentID)
		assert.Equal(t, "cpu spiked", ev.Summary)
	})

	t.Run("json wrapped in fences and prose", func(t *testing.T) {
		raw := "Here is my evidence:\n```json\n{\"success\":true,\"summary\":\"done\"}\n```"
		ev := parseEvidence("log-investigator", raw)
		assert.True(t, ev.Success)
		assert.Equal(t, "done", ev.Summary)
		// agentId defaults to the specialist name when the model omits it.
		assert.Equal(t, "log-investigator", ev.AgentID)
	})

	t.Run("unstructured output falls back to unsuccessful evidence", func(t *testing.T) {
		ev := parseEvidence("agent-x", "I could not find anything useful.")
		assert.False(t, ev.Success)
		assert.Equal(t, "agent-x", ev.AgentID)
		assert.Contains(t, ev.Summary, "unstructured")
		// The raw text is preserved (as a JSON string) so nothing is lost.
		assert.JSONEq(t, `"I could not find anything useful."`, string(ev.Data))
	})
}

func TestHandoffToolNameRoundTrip(t *testing.T) {
	name := handoffToolName("metrics-analyst")
	assert.Equal(t, "handoff_to_metrics-analyst", name)

	target, ok := parseHandoffTarget(name)
	assert.True(t, ok)
	assert.Equal(t, "metrics-analyst", target)

	_, ok = parseHandoffTarget("linsight_datasource_query")
	assert.False(t, ok)
}
