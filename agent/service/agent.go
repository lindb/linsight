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

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/agent/api"
	agentconfig "github.com/lindb/linsight/agent/config"
	modelcomp "github.com/lindb/linsight/agent/components/model"
	skillscomp "github.com/lindb/linsight/agent/components/skills"
	agentmodel "github.com/lindb/linsight/agent/model"
	"github.com/lindb/linsight/agent/service/iface"
	"github.com/lindb/linsight/agent/telemetry"
	"github.com/lindb/linsight/pkg/util"
)

var (
	agentLog = logger.GetLogger("agent", "service")

	// ErrTooManyRequests is returned when the per-org concurrency limit is exceeded.
	ErrTooManyRequests = errors.New("too many concurrent requests for this org")
)

// AgentService orchestrates LLM streaming chat with per-org model routing and SSE delivery.
type AgentService interface {
	// Chat runs one conversation turn, streaming tool and content events to writer.
	// convUID identifies an existing conversation; an empty string creates a new one.
	Chat(ctx context.Context, req *agentapi.ChatRequest, writer *agentapi.SSEWriter, convUID string) error
}

type agentService struct {
	cfg          *agentconfig.Config
	settingsSrv  iface.AgentSettingsService
	modelFactory *modelcomp.ModelFactory
	// skillRegistry applies the three-level governance funnel to select allowed tools.
	skillRegistry *skillscomp.SkillRegistry
	proxy         *modelcomp.DynamicModelProxy
	// compiled is the singleton ReAct agent graph built once at startup.
	compiled *react.Agent
	// sems maps orgID (int64) → chan struct{} (buffered semaphore).
	sems sync.Map
}

// NewAgentService creates an AgentService wired to the given dependencies.
// The ReAct graph is compiled eagerly at construction time using context.Background()
// so that any graph-build failure surfaces at startup rather than on the first request.
func NewAgentService(
	cfg *agentconfig.Config,
	settingsSrv iface.AgentSettingsService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
) (AgentService, error) {
	// Build the ReAct graph once at startup with context.Background().
	// proxy.ForOrg / proxy.WithTools are called per-request, so the graph is stateless.
	compiled, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: proxy,
		MaxStep:          cfg.MaxStep,
	})
	if err != nil {
		return nil, fmt.Errorf("compile react graph: %w", err)
	}
	return &agentService{
		cfg:           cfg,
		settingsSrv:   settingsSrv,
		modelFactory:  modelFactory,
		skillRegistry: skillRegistry,
		proxy:         proxy,
		compiled:      compiled,
	}, nil
}

// getSemaphore returns (or lazily creates) the per-org concurrency semaphore.
// The semaphore is a buffered channel of size MaxConcurrentPerOrg.
func (s *agentService) getSemaphore(orgID int64) chan struct{} {
	sem := make(chan struct{}, s.cfg.MaxConcurrentPerOrg)
	v, _ := s.sems.LoadOrStore(orgID, sem)
	return v.(chan struct{})
}

// Chat runs a full ReAct streaming conversation turn and delivers events over writer.
// It enforces a global request timeout, a per-org concurrency limit, and panic recovery.
func (s *agentService) Chat(ctx context.Context, req *agentapi.ChatRequest, writer *agentapi.SSEWriter, convUID string) (retErr error) {
	// Panic recovery: a misbehaving tool or LLM response must never kill the goroutine.
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("agent panic: %v", r)
			agentLog.Error("agent panic recovered", logger.String("panic", fmt.Sprintf("%v", r)))
			_ = writer.WriteError("internal error", panicErr)
			retErr = panicErr
		}
	}()

	// Apply a global deadline to the entire Chat() call.
	timeoutCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout.Duration())
	defer cancel()

	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID

	// Per-org concurrency guard: reject excess requests immediately.
	sem := s.getSemaphore(orgID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		return ErrTooManyRequests
	}

	// Resolve or create the conversation.
	conv, _, err := s.settingsSrv.GetOrCreateConversation(timeoutCtx, convUID)
	if err != nil {
		return fmt.Errorf("get/create conversation: %w", err)
	}

	// Persist the incoming user message before streaming starts.
	userMsg := &agentmodel.AgentMessage{
		ConversationID: conv.ID,
		Role:           agentmodel.MessageRoleUser,
		Content:        req.Content,
	}
	userMsg.CreatedBy = userID
	userMsg.UpdatedBy = userID
	if err = s.settingsSrv.SaveMessage(timeoutCtx, userMsg); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// Load recent conversation history for LLM context assembly.
	history, err := s.settingsSrv.GetRecentMessages(timeoutCtx, conv.ID, s.cfg.MemorySize)
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}

	// Build the message slice: system prompt followed by chronological history.
	messages, err := s.assembleMessages(history)
	if err != nil {
		return fmt.Errorf("assemble messages: %w", err)
	}

	// Load the governance-filtered tool set for this conversation.
	// SkillRegistry applies the three-level funnel (global → org → conversation).
	orgTools, skillMeta, err := s.skillRegistry.LoadConversationTools(timeoutCtx, orgID, conv.ID)
	if err != nil {
		agentLog.Warn("failed to load conversation tools via skill registry, proceeding without tools",
			logger.Error(err))
	}

	// Build a name→level map and inject it into context for the telemetry callback.
	metaMap := buildMetaMap(skillMeta)
	runCtx := telemetry.WithConversationID(timeoutCtx, conv.ID)
	runCtx = telemetry.WithSkillMetaMap(runCtx, metaMap)

	// Collect per-request agent options (tools + SSE callback).
	var opts []einoagent.AgentOption

	if len(orgTools) > 0 {
		toolOpts, toolErr := react.WithTools(runCtx, orgTools...)
		if toolErr != nil {
			agentLog.Warn("react.WithTools failed, proceeding without tools", logger.Error(toolErr))
		} else {
			opts = append(opts, toolOpts...)
		}
	}

	// Inject SSE callback to relay tool_call / tool_result events to the client.
	sseCallback := s.buildSSECallback(writer)
	opts = append(opts, einoagent.WithComposeOptions(compose.WithCallbacks(sseCallback)))

	// Use the pre-compiled singleton ReAct graph; tools and callbacks are injected per-request.
	reactAgent := s.compiled

	// Start streaming the LLM response.
	reader, err := reactAgent.Stream(runCtx, messages, opts...)
	if err != nil {
		_ = writer.WriteError("stream failed", err)
		return fmt.Errorf("agent stream: %w", err)
	}
	defer reader.Close()

	// Forward content chunks to the SSE client; accumulate for DB persistence.
	var sb strings.Builder
	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			_ = writer.WriteError("stream error", recvErr)
			return fmt.Errorf("stream recv: %w", recvErr)
		}
		if chunk != nil && chunk.Content != "" {
			sb.WriteString(chunk.Content)
			if writeErr := writer.WriteContent(chunk.Content); writeErr != nil {
				// Client disconnected; stop streaming but still persist what we have.
				agentLog.Debug("SSE client disconnected during stream", logger.Error(writeErr))
				break
			}
		}
	}

	// Persist the complete assistant response for future context loading.
	assistantMsg := &agentmodel.AgentMessage{
		ConversationID: conv.ID,
		Role:           agentmodel.MessageRoleAssistant,
		Content:        sb.String(),
	}
	assistantMsg.CreatedBy = userID
	assistantMsg.UpdatedBy = userID
	if saveErr := s.settingsSrv.SaveMessage(timeoutCtx, assistantMsg); saveErr != nil {
		// Log but do not fail the request — the client already received the response.
		agentLog.Warn("failed to persist assistant message", logger.Error(saveErr))
	}

	return writer.WriteDone(conv.UID)
}

// buildMetaMap converts a SkillMeta slice into a name→ControlLevel map for telemetry lookup.
func buildMetaMap(meta []skillscomp.SkillMeta) map[string]agentmodel.SkillControlLevel {
	if len(meta) == 0 {
		return nil
	}
	m := make(map[string]agentmodel.SkillControlLevel, len(meta))
	for _, sm := range meta {
		m[sm.Name] = sm.ControlLevel
	}
	return m
}

// assembleMessages builds the message slice for the LLM: system prompt first,
// then the conversation history in chronological (oldest-first) order.
func (s *agentService) assembleMessages(history []*agentmodel.AgentMessage) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0, len(history)+1)

	if s.cfg.SystemPrompt != "" {
		msgs = append(msgs, &schema.Message{
			Role:    schema.System,
			Content: s.cfg.SystemPrompt,
		})
	}

	for _, m := range history {
		msg, err := dbMessageToSchema(m)
		if err != nil {
			// Skip malformed history entries rather than aborting the entire request.
			agentLog.Warn("skipping malformed history message",
				logger.Int64("id", m.ID),
				logger.Error(err),
			)
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// dbMessageToSchema converts a persisted AgentMessage to an Eino schema.Message.
func dbMessageToSchema(m *agentmodel.AgentMessage) (*schema.Message, error) {
	msg := &schema.Message{
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
		Name:       m.ToolName,
	}

	switch m.Role {
	case agentmodel.MessageRoleSystem:
		msg.Role = schema.System
	case agentmodel.MessageRoleUser:
		msg.Role = schema.User
	case agentmodel.MessageRoleAssistant:
		msg.Role = schema.Assistant
		// Restore tool calls for accurate history replay when the assistant invoked tools.
		if m.ToolCalls != "" {
			var toolCalls []schema.ToolCall
			if err := json.Unmarshal([]byte(m.ToolCalls), &toolCalls); err != nil {
				return nil, fmt.Errorf("unmarshal tool_calls: %w", err)
			}
			msg.ToolCalls = toolCalls
		}
	case agentmodel.MessageRoleTool:
		msg.Role = schema.Tool
	default:
		return nil, fmt.Errorf("unknown message role: %q", m.Role)
	}
	return msg, nil
}

// buildSSECallback constructs an Eino callbacks.Handler that relays
// tool_call and tool_result events to the SSE client.
// ToolCallID is intentionally left empty: callbacks.RunInfo exposes only
// the tool name; the client correlates events by name and ordering.
func (s *agentService) buildSSECallback(writer *agentapi.SSEWriter) callbacks.Handler {
	toolHandler := &cbtemplate.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			_ = writer.WriteToolCall(info.Name, "", args)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			_ = writer.WriteToolResult(info.Name, "", result)
			return ctx
		},
	}
	return react.BuildAgentCallback(nil, toolHandler)
}
