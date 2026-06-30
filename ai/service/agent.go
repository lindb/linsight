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
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	agentconfig "github.com/lindb/linsight/ai/config"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	"github.com/lindb/linsight/ai/components/memory"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/ai/telemetry"
	aiutil "github.com/lindb/linsight/ai/util"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/pkg/util"
)

var agentLog = logger.GetLogger("agent", "service")

// Service orchestrates LLM streaming chat with per-org model routing and SSE delivery.
type Service interface {
	// Chat runs one conversation turn, streaming tool and content events to writer.
	// convUID identifies an existing conversation; an empty string creates a new one.
	// Returns the conversation UID (empty if the conversation was never created, e.g., semaphore rejection)
	// and any error. On error the caller is responsible for delivering the error to the client.
	Chat(ctx context.Context, req *agentapi.ChatRequest, writer *agentapi.SSEWriter, convUID string) (string, error)
}

type agentSvc struct {
	cfg          *agentconfig.Config
	convSrv      ConversationService
	modelFactory *modelcomp.ModelFactory
	// skillRegistry applies the three-level governance funnel to select allowed tools.
	skillRegistry *skillscomp.SkillRegistry
	proxy         *modelcomp.DynamicModelProxy
	// memoryStore manages conversation state persistence
	memoryStore memory.MemoryStore
	// compiled is the singleton ReAct agent graph built once at startup.
	compiled *react.Agent
	// sems maps orgID (int64) → chan struct{} (buffered semaphore).
	sems sync.Map
}

// NewService creates an Service wired to the given dependencies.
// The ReAct graph is compiled eagerly at construction time using context.Background()
// so that any graph-build failure surfaces at startup rather than on the first request.
func NewService(
	cfg *agentconfig.Config,
	convSrv ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
) (Service, error) {
	// Build the ReAct graph once at startup with context.Background().
	// proxy.ForOrg / proxy.WithTools are called per-request, so the graph is stateless.
	compiled, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: proxy,
		MaxStep:          cfg.MaxStep,
	})
	if err != nil {
		return nil, fmt.Errorf("compile react graph: %w", err)
	}
	return &agentSvc{
		cfg:           cfg,
		convSrv:       convSrv,
		modelFactory:  modelFactory,
		skillRegistry: skillRegistry,
		proxy:         proxy,
		memoryStore:   memoryStore,
		compiled:      compiled,
	}, nil
}

// getSemaphore returns (or lazily creates) the per-org concurrency semaphore.
// The semaphore is a buffered channel of size MaxConcurrentPerOrg.
func (s *agentSvc) getSemaphore(orgID int64) chan struct{} {
	// Fast path: semaphore already exists, skip allocation.
	if v, ok := s.sems.Load(orgID); ok {
		return v.(chan struct{})
	}
	sem := make(chan struct{}, s.cfg.MaxConcurrentPerOrg)
	v, _ := s.sems.LoadOrStore(orgID, sem)
	return v.(chan struct{})
}

// Chat runs a full ReAct streaming conversation turn and delivers events over writer.
// It enforces a global request timeout, a per-org concurrency limit, and panic recovery.
func (s *agentSvc) Chat(ctx context.Context, req *agentapi.ChatRequest, writer *agentapi.SSEWriter, convUID string) (createdUID string, retErr error) {
	// Panic recovery: a misbehaving tool or LLM response must never kill the goroutine.
	// Note: this defer is registered first (LIFO order means it runs last), so it runs
	// after the fallback-title defer below, which will have already seen retErr==nil.
	// Write the fallback title explicitly here as a safety net for panic scenarios.
	var isNew bool
	var conv *agentmodel.Conversation
	var userMsgSaved bool
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("agent panic: %v", r)
			agentLog.Error("agent panic recovered", logger.String("panic", fmt.Sprintf("%v", r)))
			retErr = panicErr
			// The fallback-title defer already ran with retErr==nil (LIFO); write it now.
			if isNew && conv != nil {
				s.persistFallbackTitle(ctx, conv.ID, req.Content)
			}
		}
	}()

	// Apply a global deadline to the entire Chat() call.
	timeoutCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout.Duration())
	defer cancel()
	start := time.Now()

	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID

	// Per-org concurrency guard: reject excess requests immediately.
	sem := s.getSemaphore(orgID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		return "", agentapi.ErrTooManyRequests
	}

	// Resolve or create the conversation.
	isNew = convUID == ""
	var err error
	if isNew {
		conv, err = s.convSrv.CreateConversation(timeoutCtx)
	} else {
		conv, err = s.convSrv.GetConversation(timeoutCtx, convUID)
	}
	if err != nil {
		agentLog.Error("failed to initialize conversation", logger.Error(err))
		return "", fmt.Errorf("get/create conversation: %w", err)
	}

	// For new conversations: if we return an error before generateTitle is called,
	// persist a fallback title so the sidebar never shows an empty entry.
	// Named return retErr lets the defer inspect the final return value.
	if isNew {
		defer func() {
			if retErr != nil {
				s.persistFallbackTitle(ctx, conv.ID, req.Content)
			}
		}()
	}

	// Persist the error as a conversation message when the user message was already saved
	// so the failure remains visible after a page refresh. Uses a fresh context because
	// timeoutCtx may be expired by the time this defer fires.
	defer func() {
		if retErr != nil && conv != nil && userMsgSaved {
			s.saveErrorMessage(ctx, conv.ID, userID, retErr)
		}
	}()

	// Persist the incoming user message before streaming starts.
	userMsg := &agentmodel.Message{
		ConversationID: conv.ID,
		Role:           agentmodel.MessageRoleUser,
		Content:        req.Content,
	}
	userMsg.CreatedBy = userID
	userMsg.UpdatedBy = userID
	if err = s.convSrv.SaveMessage(timeoutCtx, userMsg); err != nil {
		agentLog.Error("failed to save user message", logger.Error(err))
		return conv.UID, fmt.Errorf("save user message: %w", err)
	}
	userMsgSaved = true

	// Use MemoryStore to restore conversation state and build messages.
	// assembleErr is declared outside all branches so shadowed inner-scope errors
	// are not silently dropped after the if/else block exits.
	var messages []*schema.Message
	var assembleErr error
	if s.memoryStore != nil && s.cfg.EnableMultiTurnContext {
		agentLog.Info("memory: using MemoryStore for multi-turn context",
			logger.String("conv.uid", conv.UID),
			logger.String("promptRigidityLevel", s.cfg.PromptRigidityLevel),
		)

		state, stateErr := s.memoryStore.GetConversationState(timeoutCtx, conv.UID)
		if stateErr != nil {
			agentLog.Warn("memory: failed to restore conversation state, falling back to DB history",
				logger.String("conv.uid", conv.UID),
				logger.Error(stateErr),
			)
			// Fall back to direct DB load so the current user message is included.
			history, histErr := s.convSrv.GetRecentMessages(timeoutCtx, conv.ID, s.cfg.MemorySize)
			if histErr != nil {
				agentLog.Error("failed to load conversation history as fallback", logger.Error(histErr))
				return conv.UID, fmt.Errorf("load history fallback: %w", histErr)
			}
			messages, assembleErr = s.assembleMessages(history, req.Content)
		} else {
			agentLog.Info("memory: conversation state restored",
				logger.String("conv.uid", conv.UID),
				logger.Int("contextWindow", len(state.ContextWindow)),
				logger.Int("activeToolCalls", len(state.ActiveToolCalls)),
			)
			messages = s.buildMessagesWithMemoryState(state, req.Content)
		}
	} else {
		agentLog.Info("memory: MemoryStore disabled, using DB history",
			logger.Any("memoryStoreEnabled", s.memoryStore != nil),
			logger.Any("multiTurnContext", s.cfg.EnableMultiTurnContext),
		)
		history, histErr := s.convSrv.GetRecentMessages(timeoutCtx, conv.ID, s.cfg.MemorySize)
		if histErr != nil {
			agentLog.Error("failed to load conversation history", logger.Error(histErr))
			return conv.UID, fmt.Errorf("load history: %w", histErr)
		}
		messages, assembleErr = s.assembleMessages(history, req.Content)
	}

	if assembleErr != nil {
		agentLog.Error("failed to assemble messages", logger.Error(assembleErr))
		return conv.UID, fmt.Errorf("assemble messages: %w", assembleErr)
	}

	// Load the governance-filtered tool set for this conversation.
	// SkillRegistry applies the three-level funnel (global → org → conversation).
	orgTools, skillMeta, err := s.skillRegistry.LoadConversationTools(timeoutCtx, orgID, conv.ID)
	if err != nil {
		agentLog.Warn("failed to load conversation tools via skill registry, proceeding without tools",
			logger.Error(err))
	}

	// Note: Removed auto-generated tool calls approach as it violates OpenAI API rules
	// Assistant messages with tool_calls must be followed by tool responses
	// Instead, we rely on enhanced system prompt to guide LLM behavior

	agentLog.Info("chat session start",
		logger.Int64("org.id", orgID),
		logger.Int64("user.id", userID),
		logger.String("conv.uid", conv.UID),
		logger.Any("is_new", isNew),
		logger.Int("tools_count", len(orgTools)),
	)

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

	// toolCallsByIndex accumulates streaming fragments using the Index field defined
	// by the OpenAI streaming protocol: the first chunk carries id+name, subsequent
	// chunks carry only arguments with id="". Merging by Index (not ID) handles this.
	// toolNameToID maps tool name → call_id for SSE and DB correlation in callbacks.
	// sync.Map is used to avoid a data race between the stream loop (writer) and
	// Eino's tool-execution goroutines (readers via callback closures).
	toolCallsByIndex := make(map[int]*schema.ToolCall) // index → accumulated ToolCall
	var toolCallsIndexOrder []int                       // preserves insertion order
	var toolNameToID sync.Map                           // toolName → call_id

	// Inject SSE callback to relay tool_call / tool_result events to the client.
	// Pass &toolNameToID so the callback can correlate results with their call IDs.
	sseCallback := s.buildSSECallback(writer, conv.ID, userID, timeoutCtx, &toolNameToID)
	opts = append(opts, einoagent.WithComposeOptions(compose.WithCallbacks(sseCallback)))

	// Use the pre-compiled singleton ReAct graph; tools and callbacks are injected per-request.
	reactAgent := s.compiled

	// Start streaming the LLM response.
	reader, err := reactAgent.Stream(runCtx, messages, opts...)
	if err != nil {
		agentLog.Error("failed to start agent stream", logger.Error(err))
		return conv.UID, fmt.Errorf("agent stream: %w", err)
	}
	defer reader.Close()

	// Forward content chunks to the SSE client; accumulate for DB persistence.
	// Tool call chunks are merged by ID so each reasoning round produces exactly
	// one assistant message in the DB (not one row per streaming fragment).
	var sb strings.Builder
	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			agentLog.Error("stream recv error", logger.Error(recvErr))
			return conv.UID, fmt.Errorf("stream recv: %w", recvErr)
		}
		if chunk != nil {
			if chunk.Content != "" {
				sb.WriteString(chunk.Content)
				if writeErr := writer.WriteContent(chunk.Content); writeErr != nil {
					// Client disconnected; stop streaming but still persist what we have.
					agentLog.Debug("SSE client disconnected during stream", logger.Error(writeErr))
					break
				}
			}
			// Merge tool_call fragments by Index — OpenAI streaming sends id+name in the
			// first chunk but subsequent chunks have id="" with only arguments appended.
			// Index (not ID) is the stable per-tool identifier across chunks.
			if chunk.Role == schema.Assistant && len(chunk.ToolCalls) > 0 {
				for i := range chunk.ToolCalls {
					tc := &chunk.ToolCalls[i]
					// Determine the merge key; fall back to loop position if Index is nil.
					idx := i
					if tc.Index != nil {
						idx = *tc.Index
					}
					if existing, ok := toolCallsByIndex[idx]; ok {
						// Continuation chunk: append arguments; fill in fields if present.
						existing.Function.Arguments += tc.Function.Arguments
						if tc.ID != "" {
							existing.ID = tc.ID
						}
						if tc.Function.Name != "" {
							existing.Function.Name = tc.Function.Name
						}
						if tc.Type != "" {
							existing.Type = tc.Type
						}
					} else {
						copied := *tc
						toolCallsByIndex[idx] = &copied
						toolCallsIndexOrder = append(toolCallsIndexOrder, idx)
					}
					// Update name→ID mapping once both fields are known.
					entry := toolCallsByIndex[idx]
					if entry.Function.Name != "" && entry.ID != "" {
						toolNameToID.Store(entry.Function.Name, entry.ID)
					}
				}
			}
		}
	}

	// Persist the merged tool call round as a single assistant message so that
	// history replay sees the complete assistant→tool→assistant chain without duplicates.
	if len(toolCallsByIndex) > 0 {
		merged := make([]schema.ToolCall, 0, len(toolCallsIndexOrder))
		for _, idx := range toolCallsIndexOrder {
			merged = append(merged, *toolCallsByIndex[idx])
		}
		// Eino's FunctionCall.Arguments has omitempty; normalize empty strings to "{}"
		// so OpenAI never receives a history tool_call entry without the arguments field.
		for i := range merged {
			if merged[i].Function.Arguments == "" {
				merged[i].Function.Arguments = "{}"
			}
		}
		toolCallsJSON, _ := json.Marshal(merged)
		tcMsg := &agentmodel.Message{
			ConversationID: conv.ID,
			Role:           agentmodel.MessageRoleAssistant,
			ToolCalls:      string(toolCallsJSON),
		}
		tcMsg.CreatedBy = userID
		tcMsg.UpdatedBy = userID
		if saveErr := s.convSrv.SaveMessage(timeoutCtx, tcMsg); saveErr != nil {
			agentLog.Warn("failed to persist tool call message", logger.Error(saveErr))
		} else {
			agentLog.Info("stream: tool call message persisted",
				logger.String("conv.uid", conv.UID),
				logger.Int("tool_calls", len(merged)),
			)
		}
	}

	// Persist the complete assistant response for future context loading.
	assistantMsg := &agentmodel.Message{
		ConversationID: conv.ID,
		Role:           agentmodel.MessageRoleAssistant,
		Content:        sb.String(),
	}
	assistantMsg.CreatedBy = userID
	assistantMsg.UpdatedBy = userID
	if saveErr := s.convSrv.SaveMessage(timeoutCtx, assistantMsg); saveErr != nil {
		// Log but do not fail the request — the client already received the response.
		agentLog.Warn("failed to persist assistant message", logger.Error(saveErr))
	}

	// For new conversations: generate the title synchronously before sending done,
	// so the DB is updated and the title is included in the done event payload.
	// Uses a fresh background context so the LLM call survives even if timeoutCtx
	// has little budget left after streaming.
	var title string
	if isNew {
		title = s.generateTitle(ctx, conv.ID, req.Content)
	}
	agentLog.Info("chat session complete",
		logger.String("conv.uid", conv.UID),
		logger.Int("response_len", sb.Len()),
		logger.String("elapsed", time.Since(start).String()),
	)
	return conv.UID, writer.WriteDone(conv.UID, title)
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
func (s *agentSvc) assembleMessages(history []*agentmodel.Message, userInput string) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0, len(history)+1)

	systemPrompt := s.cfg.SystemPrompt

	// Enhance system prompt based on query intent detection
	if userInput != "" {
		queryIntent := aiutil.DetectQueryIntent(userInput)
		if queryIntent.HasQueryIntent {
			systemPrompt += " When users provide SQL queries or mention data sources, automatically use the linsight_datasource_query tool to execute the query. "
			systemPrompt += "Always call linsight_list_datasources first to get valid datasource UIDs. "
			systemPrompt += "For SQL queries, extract the query statement and data source name from the user input."
		}
	}

	if systemPrompt != "" {
		msgs = append(msgs, &schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
	}

	for _, m := range history {
		if m.Role == agentmodel.MessageRoleError {
			// error messages are for display only; exclude from LLM context
			continue
		}
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

// dbMessageToSchema converts a persisted Message to an Eino schema.Message.
func dbMessageToSchema(m *agentmodel.Message) (*schema.Message, error) {
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
			// Safety-net: normalize arguments="" to "{}" for data stored before this fix
			// and for tools that legitimately take no arguments.
			for i := range toolCalls {
				if toolCalls[i].Function.Arguments == "" {
					toolCalls[i].Function.Arguments = "{}"
				}
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

const titleGenTimeout = 10 * time.Second

// persistFallbackTitle writes a truncated user message as the conversation title.
// It uses a fresh context so the DB write succeeds even when the request context is near expiry.
// Called via defer on error paths where generateTitle was never reached.
func (s *agentSvc) persistFallbackTitle(reqCtx context.Context, convID int64, firstMessage string) {
	user := util.GetUser(reqCtx)
	titleCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	title := titleFallback(firstMessage)
	if err := s.convSrv.UpdateConversationTitle(titleCtx, convID, title); err != nil {
		agentLog.Warn("failed to persist fallback title", logger.Error(err))
	}
}

// generateTitle calls the LLM to produce a conversation title, persists it,
// and returns the title string. Falls back to a truncated first message on any
// error or empty response, so the caller always receives a non-empty title.
//
// A fresh context.Background() carrying the signed user is used so the LLM
// and DB calls succeed even when the original request context is near expiry.
func (s *agentSvc) generateTitle(reqCtx context.Context, convID int64, firstMessage string) string {
	user := util.GetUser(reqCtx)
	titleCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	var title string
	msg, err := s.proxy.Generate(titleCtx, []*schema.Message{
		{Role: schema.User, Content: buildTitlePrompt(firstMessage)},
	})
	if err != nil {
		agentLog.Warn("title generation failed, using fallback", logger.Error(err))
		title = titleFallback(firstMessage)
	} else {
		title = strings.TrimSpace(msg.Content)
		if title == "" {
			title = titleFallback(firstMessage)
		} else if runes := []rune(title); len(runes) > 100 {
			// Truncate to 100 runes to fit the DB column.
			title = string(runes[:100])
		}
	}

	if err := s.convSrv.UpdateConversationTitle(titleCtx, convID, title); err != nil {
		agentLog.Warn("failed to persist conversation title", logger.Error(err))
	}
	return title
}

// saveErrorMessage persists a MessageRoleError message so the failure is visible after refresh.
// Uses a fresh background context (like persistFallbackTitle) because timeoutCtx may be expired.
func (s *agentSvc) saveErrorMessage(reqCtx context.Context, convID int64, userID int64, err error) {
	user := util.GetUser(reqCtx)
	saveCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	code, msg := aiutil.ClassifyError(err)
	errMsg := &agentmodel.Message{
		ConversationID: convID,
		Role:           agentmodel.MessageRoleError,
		Content:        msg,
		ErrorCode:      string(code),
	}
	errMsg.CreatedBy = userID
	errMsg.UpdatedBy = userID
	if saveErr := s.convSrv.SaveMessage(saveCtx, errMsg); saveErr != nil {
		agentLog.Warn("failed to persist error message", logger.Error(saveErr))
	}
}

// titleFallback truncates msg to 40 runes as a best-effort conversation title
// when LLM title generation fails or returns an empty response.
func titleFallback(msg string) string {
	const maxRunes = 40
	s := strings.TrimSpace(msg)
	if s == "" {
		return "New Conversation"
	}
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "..."
	}
	return s
}

// buildMessagesWithMemoryState builds messages using MemoryStore conversation state.
// NOTE: SaveMessage is called before GetConversationState, so state.ContextWindow
// already contains the current user message as its last entry. Do NOT add userInput
// again — that would produce two consecutive identical user messages which confuses the LLM.
func (s *agentSvc) buildMessagesWithMemoryState(state *memory.ConversationState, userInput string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(state.ContextWindow)+1)

	// Apply the same query-intent enhancement as assembleMessages so the system prompt
	// hints the LLM to call datasource tools when a query is detected.
	systemPrompt := s.cfg.SystemPrompt
	if userInput != "" {
		queryIntent := aiutil.DetectQueryIntent(userInput)
		if queryIntent.HasQueryIntent {
			systemPrompt += " When users provide SQL queries or mention data sources, automatically use the linsight_datasource_query tool to execute the query. "
			systemPrompt += "Always call linsight_list_datasources first to get valid datasource UIDs. "
			systemPrompt += "For SQL queries, extract the query statement and data source name from the user input."
		}
	}

	// Add system prompt
	if systemPrompt != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
		prompt := systemPrompt
		if len(prompt) > 80 {
			prompt = prompt[:80] + "..."
		}
		agentLog.Info("prompt: system prompt injected",
			logger.String("conv.uid", state.ConversationID),
			logger.String("preview", prompt),
			logger.Int("len", len(systemPrompt)),
		)
	}

	// Add conversation history from context window.
	// The last entry is already the current user message (saved to DB before this call).
	for _, msg := range state.ContextWindow {
		messages = append(messages, msg)
	}

	agentLog.Info("prompt: final message list built",
		logger.String("conv.uid", state.ConversationID),
		logger.Int("total_messages", len(messages)),
		logger.Int("history_messages", len(state.ContextWindow)),
	)

	return messages
}

// buildTitlePrompt constructs the prompt sent to the LLM for title generation.
// The first user message is trimmed to 500 runes to stay within token budgets
// and to avoid splitting multibyte characters (e.g. CJK).
// The LLM is explicitly instructed to match the language of the user message so
// that English prompts produce English titles and Chinese prompts produce Chinese titles.
func buildTitlePrompt(msg string) string {
	const maxMsgRunes = 500
	runes := []rune(msg)
	if len(runes) > maxMsgRunes {
		msg = string(runes[:maxMsgRunes])
	}
	return fmt.Sprintf(
		"Generate a concise title (5-10 words) for a conversation that starts with the following user message.\n"+
			"Reply with only the title — no quotes, no trailing punctuation, no extra text.\n"+
			"IMPORTANT: The title MUST be written in the SAME language as the user message below.\n\n"+
			"User message: %s",
		msg,
	)
}

// buildSSECallback constructs an Eino callbacks.Handler that relays
// tool_call and tool_result events to the SSE client and persists tool
// results to the DB for accurate history replay.
// convID, userID identify the current conversation and user; saveCtx is the
// request-scoped context used for DB writes (still alive during streaming).
// toolNameToID is a sync.Map populated by the stream loop; it maps tool name →
// LLM-generated call ID. Using sync.Map prevents a data race when Eino's
// tool-execution goroutines read the map while the stream loop writes to it.
func (s *agentSvc) buildSSECallback(writer *agentapi.SSEWriter, convID int64, userID int64, saveCtx context.Context, toolNameToID *sync.Map) callbacks.Handler {
	toolHandler := &cbtemplate.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			callID := ""
			if v, ok := toolNameToID.Load(info.Name); ok {
				callID = v.(string)
			}
			_ = writer.WriteToolCall(info.Name, callID, args)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			callID := ""
			if v, ok := toolNameToID.Load(info.Name); ok {
				callID = v.(string)
			}
			_ = writer.WriteToolResult(info.Name, callID, result)

			// Persist tool result so future history replay includes the full
			// assistant→tool→assistant chain, preventing the LLM from repeating
			// verbal-promise responses when context is loaded from DB.
			toolMsg := &agentmodel.Message{
				ConversationID: convID,
				Role:           agentmodel.MessageRoleTool,
				Content:        result,
				ToolName:       info.Name,
				ToolCallID:     callID,
			}
			toolMsg.CreatedBy = userID
			toolMsg.UpdatedBy = userID
			if err := s.convSrv.SaveMessage(saveCtx, toolMsg); err != nil {
				agentLog.Warn("failed to persist tool result",
					logger.String("tool", info.Name),
					logger.Error(err),
				)
			} else {
				agentLog.Info("stream: tool result persisted",
					logger.String("tool", info.Name),
					logger.Int("result_len", len(result)),
				)
			}
			return ctx
		},
	}
	return react.BuildAgentCallback(nil, toolHandler)
}
