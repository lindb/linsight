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

// Package multiagent implements a Host-Specialist multi-agent layer on top of the
// existing agent runtime. Each DB-configured Specialist is exposed to the Host LLM
// as a handoff_to_<name> tool; invoking it runs the specialist in an isolated context
// and returns a structured Evidence object that flows back to the Host as a tool result.
package multiagent

import (
	"encoding/json"
	"strings"
)

// evidenceContract is appended to every specialist's system prompt to force a
// machine-parseable handback. The Host reads the returned evidence (as a tool result)
// and produces the user-facing summary — the specialist never addresses the user directly.
const evidenceContract = `

# Output Contract (MANDATORY)
You are a specialist sub-agent. Do NOT address the end user. When you finish, respond with
EXACTLY ONE JSON object and NOTHING else — no prose, no markdown, no code fences. Schema:
{
  "agentId": "<your agent name>",
  "success": true | false,
  "summary": "<concise natural-language conclusion for the host>",
  "data": <optional JSON value with structured findings>,
  "error": "<present only when success is false>"
}`

// Evidence is the structured object a specialist hands back to the Host. It is the
// single unit of "evidence handback": the specialist is destroyed after producing it and
// control returns to the Host, which reads Summary/Data to write the final answer.
type Evidence struct {
	// AgentID is the specialist name that produced this evidence.
	AgentID string `json:"agentId"`
	// Success reports whether the specialist completed its goal.
	Success bool `json:"success"`
	// Summary is a concise natural-language conclusion for the host.
	Summary string `json:"summary"`
	// Data carries optional structured findings.
	Data json.RawMessage `json:"data,omitempty"`
	// Error is set only when Success is false.
	Error string `json:"error,omitempty"`
}

// marshal serializes the evidence to a compact JSON string for use as a tool result.
// On the (practically impossible) marshal failure it falls back to a minimal object so
// the tool layer always returns valid JSON to the host.
func (e *Evidence) marshal() string {
	b, err := json.Marshal(e)
	if err != nil {
		return `{"success":false,"summary":"failed to marshal evidence","error":"marshal error"}`
	}
	return string(b)
}

// failureEvidence builds the JSON tool result for a failed handoff. The tool layer never
// returns a Go error — a failure is encoded as unsuccessful evidence so the Host decides
// how to proceed.
func failureEvidence(agentID, summary, errMsg string) string {
	ev := Evidence{AgentID: agentID, Success: false, Summary: summary, Error: errMsg}
	return ev.marshal()
}

// parseEvidence turns a specialist's raw LLM output into an Evidence. It is deliberately
// lenient: the specialist is instructed to emit JSON-only, but models occasionally wrap it
// in prose or code fences. When no valid JSON object can be recovered, the raw text is
// preserved as an unstructured, unsuccessful evidence so the host still sees the output.
func parseEvidence(agentID, raw string) Evidence {
	if js := extractJSON(raw); js != nil {
		var ev Evidence
		if err := json.Unmarshal(js, &ev); err == nil {
			if ev.AgentID == "" {
				ev.AgentID = agentID
			}
			return ev
		}
	}
	// Fallback: keep the raw text so nothing is silently lost.
	rawJSON, _ := json.Marshal(strings.TrimSpace(raw))
	return Evidence{
		AgentID: agentID,
		Success: false,
		Summary: "specialist returned unstructured output",
		Data:    rawJSON,
	}
}

// extractJSON returns the first balanced top-level JSON object embedded in s, stripping
// optional ```json / ``` code fences first. Returns nil when no candidate object is found.
// It respects string literals and escapes so braces inside strings do not unbalance the scan.
func extractJSON(s string) []byte {
	s = stripCodeFences(s)

	start := strings.IndexByte(s, '{')
	if start < 0 {
		return nil
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return []byte(s[start : i+1])
			}
		}
	}
	return nil
}

// stripCodeFences removes a leading ```json (or ```) fence and its trailing ``` when the
// trimmed content is wrapped in a Markdown code block. Content without fences is returned as-is.
func stripCodeFences(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return s
	}
	// Drop the first fence line (```/```json) entirely.
	if nl := strings.IndexByte(t, '\n'); nl >= 0 {
		t = t[nl+1:]
	} else {
		return s
	}
	if idx := strings.LastIndex(t, "```"); idx >= 0 {
		t = t[:idx]
	}
	return t
}
