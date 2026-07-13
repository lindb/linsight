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

package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"
)

// toolName is the tool name registered in the agent's tool set and invoked by the LLM.
const toolName = "skill"

// toolDescriptionBase is the instruction preamble injected before the available skills list.
const toolDescriptionBase = `Execute a skill within the main conversation

<skills_instructions>
When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively. Skills provide specialized capabilities and domain knowledge.

How to invoke:
- Use the exact string inside <name> tag as the skill name (no arguments)
- Example: ` + "`skill: \"my-skill\"`" + `

Important:
- When a skill is relevant, you must invoke this tool IMMEDIATELY as your first action
- NEVER just announce or mention a skill in your text response without actually calling this tool
- This is a BLOCKING REQUIREMENT: invoke the relevant skill tool BEFORE generating any other response about the task
- Only use skills listed in <available_skills> below
- Do not invoke a skill that is already running
</skills_instructions>
`

// toolResultFmt is the format string for inline mode: tool call acknowledgment
// followed by the skill content.
const toolResultFmt = "Launching skill: %s\n\n%s"

// forkResultFmt is the format string returned after a fork/fork_with_context execution.
const forkResultFmt = "Skill %q completed (sub-agent execution).\n\nResult:\n%s"

// availableSkillsTemplate renders the <available_skills> block in the tool description.
var availableSkillsTemplate = template.Must(template.New("skills").Parse(`
<available_skills>
{{- range . }}
<skill>
<name>
{{ .Name }}
</name>
<description>
{{ .Description }}
</description>
</skill>
{{- end }}
</available_skills>
`))

// inputArguments is the JSON schema for the LLM's tool call.
type inputArguments struct {
	Skill string `json:"skill"`
}

// nativeSkillTool implements tool.BaseTool. It delegates to a Backend which
// handles org scoping internally via ctx.
type nativeSkillTool struct {
	backend Backend
}

// NewSkillTool creates a tool.BaseTool that lets the LLM invoke skills via
// `{"skill": "<name>"}`. org scoping and governance are handled by the Backend.
func NewSkillTool(backend Backend) tool.BaseTool {
	return &nativeSkillTool{backend: backend}
}

// Info renders the tool description including the <available_skills> list.
func (t *nativeSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	matters, err := t.backend.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("skill tool: list skills: %w", err)
	}

	log.Debug("skill tool: rendering available skills for LLM",
		logger.Int("count", len(matters)))

	var buf bytes.Buffer
	if err := availableSkillsTemplate.Execute(&buf, matters); err != nil {
		return nil, fmt.Errorf("skill tool: render available skills: %w", err)
	}

	return &schema.ToolInfo{
		Name: toolName,
		Desc: toolDescriptionBase + buf.String(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill": {
				Type:     schema.String,
				Desc:     "The skill name (no arguments). E.g., \"my-skill\"",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the skill. It loads the full skill definition from the
// backend and dispatches by ContextMode:
//   - inline (""): returns the skill Content directly
//   - fork / fork_with_context: TODO — requires sub-agent infrastructure
func (t *nativeSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := &inputArguments{}
	if err := json.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", fmt.Errorf("skill tool: unmarshal arguments: %w", err)
	}
	if args.Skill == "" {
		return "", fmt.Errorf("skill tool: skill name is required")
	}

	sk, err := t.backend.Get(ctx, args.Skill)
	if err != nil {
		log.Warn("skill tool: skill not found or not authorized",
			logger.String("skill", args.Skill), logger.Error(err))
		return "", fmt.Errorf("skill tool: %w", err)
	}

	log.Info("skill tool: executing skill",
		logger.String("skill", sk.Name),
		logger.Any("context_mode", string(sk.Context)),
		logger.Int("content_len", len(sk.Content)),
	)

	switch sk.Context {
	case skill.ContextModeForkWithContext, skill.ContextModeFork:
		log.Warn("skill tool: fork mode not yet wired; falling back to inline",
			logger.String("skill", sk.Name), logger.Any("context_mode", sk.Context))
		fallthrough
	default:
		result := fmt.Sprintf(toolResultFmt, sk.Name, sk.Content)
		log.Debug("skill tool: inline execution done",
			logger.String("skill", sk.Name), logger.Int("result_len", len(result)))
		return result, nil
	}
}
