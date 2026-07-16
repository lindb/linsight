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
	"fmt"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"
)

var handoffLog = logger.GetLogger("agent", "multiagent")

// specialistGenerator is the minimal model surface a handoff tool needs to run its
// specialist. *modelcomp.DynamicModelProxy satisfies it, so production wiring uses the
// real per-org model; tests inject a stub. Kept local to avoid coupling to the proxy type.
type specialistGenerator interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
}

// handoffToolPrefix is the tool-name prefix that marks a handoff-to-specialist tool.
const handoffToolPrefix = "handoff_to_"

// handoffToolName returns the handoff tool name for a specialist by its slug name.
func handoffToolName(name string) string {
	return handoffToolPrefix + name
}

// parseHandoffTarget extracts the specialist name from a handoff tool name.
// Returns ("", false) when toolName is not a handoff tool.
func parseHandoffTarget(toolName string) (string, bool) {
	if !strings.HasPrefix(toolName, handoffToolPrefix) {
		return "", false
	}
	return strings.TrimPrefix(toolName, handoffToolPrefix), true
}

// handoffArgs is the JSON schema for the Host LLM's handoff tool call.
type handoffArgs struct {
	// Goal is the self-contained, pruned task the Host delegates to the specialist.
	Goal string `json:"goal"`
}

// handoffTool is the runtime realization of a Specialist: an Eino InvokableTool that,
// when called by the Host, runs the specialist in an ISOLATED context and hands back a
// structured Evidence object. This single type embodies three of the four Claude-Code
// pillars — the dynamic specialist executor, context isolation, and evidence handback.
type handoffTool struct {
	spec  *runtimeSpecialist
	proxy specialistGenerator
}

// newHandoffTool builds a handoff tool for the given specialist.
func newHandoffTool(spec *runtimeSpecialist, proxy specialistGenerator) *handoffTool {
	return &handoffTool{spec: spec, proxy: proxy}
}

// Info exposes the specialist to the Host LLM as a handoff_to_<name> function whose
// description is the specialist's IntendedUse (so the model knows when to delegate).
func (t *handoffTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	desc := t.spec.Description
	if strings.TrimSpace(desc) == "" {
		desc = fmt.Sprintf("Delegate a self-contained sub-task to the %q specialist.", t.spec.Name)
	}
	return &schema.ToolInfo{
		Name: handoffToolName(t.spec.Name),
		Desc: desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"goal": {
				Type: schema.String,
				Desc: "The self-contained, fully-specified task goal for the specialist. " +
					"Include ALL context the specialist needs; it cannot see the conversation history.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the specialist and returns its evidence as the tool result.
//
//	① parse goal        — the Host's pruned, self-contained objective
//	② context isolation — rebuild a CLEAN message slice (specialist system prompt +
//	                       evidence contract, then the goal); global history is discarded
//	③ evidence handback — call the per-org model, strong-parse the reply into Evidence,
//	                       and return it as JSON. Errors never propagate: they become an
//	                       unsuccessful Evidence so the Host decides how to proceed.
func (t *handoffTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args handoffArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return failureEvidence(t.spec.Name, "invalid handoff arguments", err.Error()), nil
	}
	if strings.TrimSpace(args.Goal) == "" {
		return failureEvidence(t.spec.Name, "no goal supplied to specialist", "handoff goal is required"), nil
	}

	// Context isolation: the specialist sees ONLY its own persona + the goal.
	cleanMessages := []*schema.Message{
		{Role: schema.System, Content: t.spec.SystemPrompt + evidenceContract},
		{Role: schema.User, Content: args.Goal},
	}

	handoffLog.Info("multiagent: dispatching specialist",
		logger.String("specialist", t.spec.Name),
		logger.Int("goal_len", len(args.Goal)),
	)

	// The proxy resolves the per-org model from the signed user in ctx (real routing).
	out, err := t.proxy.Generate(ctx, cleanMessages)
	if err != nil {
		handoffLog.Warn("multiagent: specialist generation failed",
			logger.String("specialist", t.spec.Name), logger.Error(err))
		return failureEvidence(t.spec.Name, "specialist execution failed", err.Error()), nil
	}

	ev := parseEvidence(t.spec.Name, out.Content)
	handoffLog.Info("multiagent: specialist handback",
		logger.String("specialist", t.spec.Name),
		logger.Any("success", ev.Success),
	)
	return ev.marshal(), nil
}
