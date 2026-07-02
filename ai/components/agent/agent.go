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

// Package agent provides the ReAct streaming agent runtime.
// It is intentionally separate from ai/service/ (which handles management CRUD)
// to keep streaming orchestration concerns isolated.
package agent

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

	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentmodel "github.com/lindb/linsight/ai/model"
	agentservice "github.com/lindb/linsight/ai/service"
	"github.com/lindb/linsight/ai/telemetry"
	aiutil "github.com/lindb/linsight/ai/util"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

var agentLog = logger.GetLogger("agent", "runtime")

// Agent orchestrates LLM streaming chat with per-org model routing and AG-UI SSE delivery.
type Agent interface {
	// Chat runs one conversation turn, streaming AG-UI events to writer.
	// convUID identifies an existing conversation; an empty string creates a new one.
	// Returns the conversation UID, the run ID (always set, even on early failure), and any error.
	Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (string, string, error)
}

type agentRuntime struct {
	cfg           *agentconfig.Config
	convSrv       agentservice.ConversationService
	modelFactory  *modelcomp.ModelFactory
	skillRegistry *skillscomp.SkillRegistry
	proxy         *modelcomp.DynamicModelProxy
	memoryStore   memory.MemoryStore
	// compiled is the singleton ReAct agent graph built once at startup.
	compiled *react.Agent
	// sems maps orgID (int64) → chan struct{} (buffered semaphore).
	sems sync.Map
}

// fullResponseStreamToolCallChecker reads all streaming chunks to detect tool calls.
// Unlike the default firstChunkStreamToolCallChecker, this handles LLMs (e.g. DeepSeek, Claude)
// that emit text content before tool_calls in the same streaming response.
// The handler closes sr before returning as required by the eino contract.
func fullResponseStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

// New creates an Agent wired to the given dependencies.
// The ReAct graph is compiled eagerly at construction time using context.Background()
// so that any graph-build failure surfaces at startup rather than on the first request.
func New(
	cfg *agentconfig.Config,
	convSrv agentservice.ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
) (Agent, error) {
	compiled, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel:      proxy,
		MaxStep:               cfg.MaxStep,
		StreamToolCallChecker: fullResponseStreamToolCallChecker,
		ToolsConfig: compose.ToolsNodeConfig{
			// When the LLM hallucinates a non-existent tool name, return a descriptive
			// error message instead of crashing the run so the LLM can self-correct.
			UnknownToolsHandler: func(_ context.Context, name, _ string) (string, error) {
				return fmt.Sprintf("tool %q not found; available tools are listed in the system prompt", name), nil
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("compile react graph: %w", err)
	}
	return &agentRuntime{
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
func (a *agentRuntime) getSemaphore(orgID int64) chan struct{} {
	if v, ok := a.sems.Load(orgID); ok {
		return v.(chan struct{})
	}
	sem := make(chan struct{}, a.cfg.MaxConcurrentPerOrg)
	v, _ := a.sems.LoadOrStore(orgID, sem)
	return v.(chan struct{})
}

// Chat runs a full ReAct streaming conversation turn and delivers AG-UI events over writer.
// It enforces a global request timeout, a per-org concurrency limit, and panic recovery.
func (a *agentRuntime) Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (createdUID string, runID string, retErr error) {
	var isNew bool
	var conv *agentmodel.Conversation
	var userMsgUID string

	// Generate runID immediately so it is always available for WriteRunError,
	// even when the function returns before WriteRunStarted is emitted.
	runID = uuid.GenerateShortUUID()

	// Panic recovery: must never kill the goroutine.
	// Registered first (LIFO) so it runs last, after the fallback-title defer.
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("agent panic: %v", r)
			agentLog.Error("agent panic recovered", logger.String("panic", fmt.Sprintf("%v", r)))
			retErr = panicErr
			if isNew && conv != nil {
				a.persistFallbackTitle(ctx, conv.ID, req.Content)
			}
		}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, a.cfg.RequestTimeout.Duration())
	defer cancel()
	start := time.Now()

	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID

	// Per-org concurrency guard.
	sem := a.getSemaphore(orgID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		return "", runID, agentapi.ErrTooManyRequests
	}

	// Resolve or create the conversation.
	isNew = convUID == ""
	var err error
	if isNew {
		conv, err = a.convSrv.CreateConversation(timeoutCtx)
	} else {
		conv, err = a.convSrv.GetConversation(timeoutCtx, convUID)
	}
	if err != nil {
		agentLog.Error("failed to initialize conversation", logger.Error(err))
		return "", runID, fmt.Errorf("get/create conversation: %w", err)
	}

	// Emit RUN_STARTED so the client knows this run is live.
	writer.WriteRunStarted(runID, conv.UID)

	// For new conversations: persist a fallback title if we return with an error
	// so the sidebar never shows an empty entry.
	if isNew {
		defer func() {
			if retErr != nil {
				a.persistFallbackTitle(ctx, conv.ID, req.Content)
			}
		}()
	}

	// Persist the error message so the failure is visible after page refresh.
	defer func() {
		if retErr != nil && conv != nil && userMsgUID != "" {
			a.saveErrorMessage(ctx, conv.ID, conv.UID, userID, runID, retErr)
		}
	}()

	// Pre-generate the assistant message UID so it is known before streaming starts.
	// This lets tool-call events reference the parent message ID immediately.
	assistantMsgUID := uuid.GenerateShortUUID()

	// Persist the incoming user message.
	userMsgUID = uuid.GenerateShortUUID()
	userMsg := &schema.Message{
		Role:    schema.User,
		Content: req.Content,
	}
	rawUser, _ := json.Marshal(userMsg)
	dbUserMsg := &agentmodel.Message{
		ConversationID:  conv.ID,
		ConversationUID: conv.UID,
		MessageUID:      userMsgUID,
		Role:            schema.User,
		RawMessage:      json.RawMessage(rawUser),
		TurnUID:         runID,
		RunUID:          runID,
	}
	dbUserMsg.CreatedBy = userID
	dbUserMsg.UpdatedBy = userID
	if err = a.convSrv.SaveMessage(timeoutCtx, dbUserMsg); err != nil {
		agentLog.Error("failed to save user message", logger.Error(err))
		return conv.UID, runID, fmt.Errorf("save user message: %w", err)
	}

	// Build LLM message list from memory store or DB history.
	messages, assembleErr := a.buildLLMMessages(timeoutCtx, conv, req.Content)
	if assembleErr != nil {
		agentLog.Error("failed to build LLM messages", logger.Error(assembleErr))
		return conv.UID, runID, fmt.Errorf("build messages: %w", assembleErr)
	}

	// Load the governance-filtered tool set for this conversation.
	orgTools, skillMeta, err := a.skillRegistry.LoadConversationTools(timeoutCtx, orgID, conv.ID)
	if err != nil {
		agentLog.Warn("failed to load conversation tools, proceeding without tools", logger.Error(err))
	}

	agentLog.Info("chat session start",
		logger.Int64("org.id", orgID),
		logger.Int64("user.id", userID),
		logger.String("conv.uid", conv.UID),
		logger.Any("is_new", isNew),
		logger.Int("tools_count", len(orgTools)),
	)

	metaMap := buildMetaMap(skillMeta)
	runCtx := telemetry.WithConversationID(timeoutCtx, conv.ID)
	runCtx = telemetry.WithSkillMetaMap(runCtx, metaMap)

	var opts []einoagent.AgentOption
	if len(orgTools) > 0 {
		toolOpts, toolErr := react.WithTools(runCtx, orgTools...)
		if toolErr != nil {
			agentLog.Warn("react.WithTools failed, proceeding without tools", logger.Error(toolErr))
		} else {
			opts = append(opts, toolOpts...)
		}
	}

	// toolCallsByIndex accumulates streaming fragments keyed by Index field.
	toolCallsByIndex := make(map[int]*schema.ToolCall)
	var toolCallsIndexOrder []int

	// sb and textStarted are declared before buildSSECallback so that
	// ModelCallbackHandler.OnError can capture them by pointer.
	var textStarted bool
	var sb strings.Builder

	sseCallback := a.buildSSECallback(writer, conv.ID, conv.UID, userID, timeoutCtx, assistantMsgUID,
		runID, toolCallsByIndex, &toolCallsIndexOrder, &sb)
	opts = append(opts, einoagent.WithComposeOptions(compose.WithCallbacks(sseCallback)))

	reader, err := a.compiled.Stream(runCtx, messages, opts...)
	if err != nil {
		agentLog.Error("failed to start agent stream", logger.Error(err))
		return conv.UID, runID, fmt.Errorf("agent stream: %w", err)
	}
	defer reader.Close()

	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			agentLog.Error("stream recv error", logger.Error(recvErr))
			return conv.UID, runID, fmt.Errorf("stream recv: %w", recvErr)
		}
		if chunk == nil {
			continue
		}

		if chunk.Content != "" {
			if !textStarted {
				writer.WriteTextMessageStart(assistantMsgUID, schema.Assistant)
				textStarted = true
			}
			sb.WriteString(chunk.Content)
			writer.WriteTextMessageContent(assistantMsgUID, chunk.Content)
		}

		// Merge tool_call fragments by Index (OpenAI streaming protocol).
		if chunk.Role == schema.Assistant && len(chunk.ToolCalls) > 0 {
			for i := range chunk.ToolCalls {
				tc := &chunk.ToolCalls[i]
				idx := i
				if tc.Index != nil {
					idx = *tc.Index
				}
				if existing, ok := toolCallsByIndex[idx]; ok {
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
			}
		}
	}

	if textStarted {
		writer.WriteTextMessageEnd(assistantMsgUID)
	}

	// Persist the merged assistant message (content + tool_calls in one row).
	a.savePartialAssistantMessage(timeoutCtx, conv.ID, conv.UID, userID,
		assistantMsgUID, sb.String(), runID, toolCallsByIndex, toolCallsIndexOrder)

	// Generate a title for new conversations and emit RUN_FINISHED.
	var title string
	if isNew {
		title = a.generateTitle(ctx, conv.ID, req.Content)
	}

	agentLog.Info("chat session complete",
		logger.String("conv.uid", conv.UID),
		logger.Int("response_len", sb.Len()),
		logger.String("elapsed", time.Since(start).String()),
	)

	writer.WriteRunFinished(runID, conv.UID, title)
	return conv.UID, runID, nil
}

// buildLLMMessages assembles the []*schema.Message for the LLM from either the
// MemoryStore (if enabled) or a direct DB history load.
func (a *agentRuntime) buildLLMMessages(ctx context.Context, conv *agentmodel.Conversation, userInput string) ([]*schema.Message, error) {
	var messages []*schema.Message

	if a.memoryStore != nil && a.cfg.EnableMultiTurnContext {
		agentLog.Info("memory: using MemoryStore for multi-turn context",
			logger.String("conv.uid", conv.UID))

		state, stateErr := a.memoryStore.GetConversationState(ctx, conv.UID)
		if stateErr != nil {
			agentLog.Warn("memory: failed to restore conversation state, falling back to DB history",
				logger.String("conv.uid", conv.UID), logger.Error(stateErr))
			history, histErr := a.convSrv.GetRecentMessages(ctx, conv.ID, a.cfg.MemorySize)
			if histErr != nil {
				return nil, fmt.Errorf("load history fallback: %w", histErr)
			}
			var assembleErr error
			messages, assembleErr = a.assembleMessages(history, userInput)
			if assembleErr != nil {
				return nil, assembleErr
			}
		} else {
			agentLog.Info("memory: conversation state restored",
				logger.String("conv.uid", conv.UID),
				logger.Int("contextWindow", len(state.ContextWindow)))
			messages = a.buildMessagesWithMemoryState(state, userInput)
		}
	} else {
		agentLog.Info("memory: MemoryStore disabled, using DB history")
		history, histErr := a.convSrv.GetRecentMessages(ctx, conv.ID, a.cfg.MemorySize)
		if histErr != nil {
			return nil, fmt.Errorf("load history: %w", histErr)
		}
		var assembleErr error
		messages, assembleErr = a.assembleMessages(history, userInput)
		if assembleErr != nil {
			return nil, assembleErr
		}
	}
	return messages, nil
}

// assembleMessages builds the LLM message slice from DB history rows.
// RawMessage stores eino schema.Message JSON (snake_case keys), so direct
// unmarshal to schema.Message is lossless — ToolCalls, ToolCallID, and ToolName
// are preserved exactly.
func (a *agentRuntime) assembleMessages(history []*agentmodel.Message, userInput string) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0, len(history)+1)

	systemPrompt := a.buildSystemPrompt(userInput)
	if systemPrompt != "" {
		msgs = append(msgs, &schema.Message{Role: schema.System, Content: systemPrompt})
	}

	// Collect all tool call_ids referenced by assistant messages so we can detect
	// orphaned tool-result rows (those whose call_id has no matching parent).
	knownCallIDs := make(map[string]bool)
	for _, m := range history {
		if m.Role != schema.Assistant {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			continue
		}
		for _, tc := range sm.ToolCalls {
			if tc.ID != "" {
				knownCallIDs[tc.ID] = true
			}
		}
	}

	for _, m := range history {
		if m.Role == agentmodel.RoleError {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			agentLog.Warn("skipping malformed history message",
				logger.Int64("id", m.ID), logger.Error(err))
			continue
		}
		// Drop orphaned tool messages; sending them to the LLM causes an API validation
		// error because their ToolCallID references a non-existent parent ToolCall.
		if sm.Role == schema.Tool && !knownCallIDs[sm.ToolCallID] {
			agentLog.Warn("skipping orphaned tool message",
				logger.Int64("id", m.ID), logger.String("tool_call_id", sm.ToolCallID))
			continue
		}
		msgs = append(msgs, &sm)
	}
	return msgs, nil
}

// buildMessagesWithMemoryState builds messages using MemoryStore conversation state.
// The state's ContextWindow already contains the current user message (saved before
// GetConversationState was called), so we must NOT append userInput again.
func (a *agentRuntime) buildMessagesWithMemoryState(state *memory.ConversationState, userInput string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(state.ContextWindow)+1)

	systemPrompt := a.buildSystemPrompt(userInput)
	if systemPrompt != "" {
		messages = append(messages, &schema.Message{Role: schema.System, Content: systemPrompt})
		preview := systemPrompt
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		agentLog.Info("prompt: system prompt injected",
			logger.String("conv.uid", state.ConversationID),
			logger.String("preview", preview))
	}

	for _, msg := range state.ContextWindow {
		messages = append(messages, msg)
	}

	agentLog.Info("prompt: final message list built",
		logger.String("conv.uid", state.ConversationID),
		logger.Int("total_messages", len(messages)),
		logger.Int("history_messages", len(state.ContextWindow)))

	return messages
}

// buildSystemPrompt returns the base system prompt, optionally enhanced with
// datasource-query hints when the user input looks like a data query.
func (a *agentRuntime) buildSystemPrompt(userInput string) string {
	p := a.cfg.SystemPrompt
	// if userInput != "" {
	// 	qi := aiutil.DetectQueryIntent(userInput)
	// 	if qi.HasQueryIntent {
	// 		p += " When users provide SQL queries or mention data sources, automatically use the linsight_datasource_query tool to execute the query. "
	// 		p += "Always call linsight_list_datasources first to get valid datasource UIDs. "
	// 		p += "For SQL queries, extract the query statement and data source name from the user input."
	// 	}
	// }
	return p
}

// buildMetaMap converts a SkillMeta slice into a name→ControlLevel map for telemetry.
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

const titleGenTimeout = 10 * time.Second

// persistFallbackTitle writes a truncated user message as the conversation title.
func (a *agentRuntime) persistFallbackTitle(reqCtx context.Context, convID int64, firstMessage string) {
	user := util.GetUser(reqCtx)
	titleCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	title := titleFallback(firstMessage)
	if err := a.convSrv.UpdateConversationTitle(titleCtx, convID, title); err != nil {
		agentLog.Warn("failed to persist fallback title", logger.Error(err))
	}
}

// generateTitle calls the LLM for a conversation title, persists it, and returns it.
// Falls back to a truncated first message on any error so the caller always gets
// a non-empty string.
func (a *agentRuntime) generateTitle(reqCtx context.Context, convID int64, firstMessage string) string {
	user := util.GetUser(reqCtx)
	titleCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	var title string
	msg, err := a.proxy.Generate(titleCtx, []*schema.Message{
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
			title = string(runes[:100])
		}
	}

	if err := a.convSrv.UpdateConversationTitle(titleCtx, convID, title); err != nil {
		agentLog.Warn("failed to persist conversation title", logger.Error(err))
	}
	return title
}

// savePartialAssistantMessage persists an assistant message that may contain partial
// content and/or tool calls. It is called both at the normal end of streaming and by
// ModelCallbackHandler.OnError to guarantee that tool-result rows always have a parent.
func (a *agentRuntime) savePartialAssistantMessage(
	ctx context.Context,
	convID int64, convUID string, userID int64,
	assistantMsgUID string, content string, turnUID string,
	toolCallsByIndex map[int]*schema.ToolCall, toolCallsIndexOrder []int,
) {
	var mergedToolCalls []schema.ToolCall
	for _, idx := range toolCallsIndexOrder {
		tc := *toolCallsByIndex[idx]
		// Normalize empty arguments to "{}" — OpenAI requires valid JSON.
		if tc.Function.Arguments == "" {
			tc.Function.Arguments = "{}"
		}
		mergedToolCalls = append(mergedToolCalls, tc)
	}
	msg := &schema.Message{
		Role:      schema.Assistant,
		Content:   content,
		ToolCalls: mergedToolCalls,
	}
	raw, _ := json.Marshal(msg)
	dbMsg := &agentmodel.Message{
		ConversationID:  convID,
		ConversationUID: convUID,
		MessageUID:      assistantMsgUID,
		Role:            schema.Assistant,
		RawMessage:      json.RawMessage(raw),
		TurnUID:         turnUID,
		RunUID:          turnUID,
	}
	dbMsg.CreatedBy = userID
	dbMsg.UpdatedBy = userID
	if err := a.convSrv.SaveMessage(ctx, dbMsg); err != nil {
		agentLog.Warn("failed to persist assistant message", logger.Error(err))
	}
}

// saveErrorMessage persists a RoleError message so failures are visible after refresh.
func (a *agentRuntime) saveErrorMessage(reqCtx context.Context, convID int64, convUID string, userID int64, turnUID string, chatErr error) {
	user := util.GetUser(reqCtx)
	saveCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	_, errMsg := aiutil.ClassifyError(chatErr)
	msgUID := uuid.GenerateShortUUID()
	errSchemaMsg := &schema.Message{
		Role:    agentmodel.RoleError,
		Content: errMsg,
	}
	raw, _ := json.Marshal(errSchemaMsg)
	dbMsg := &agentmodel.Message{
		ConversationID:  convID,
		ConversationUID: convUID,
		MessageUID:      msgUID,
		Role:            agentmodel.RoleError,
		RawMessage:      json.RawMessage(raw),
		TurnUID:         turnUID,
		RunUID:          turnUID,
	}
	dbMsg.CreatedBy = userID
	dbMsg.UpdatedBy = userID
	if saveErr := a.convSrv.SaveMessage(saveCtx, dbMsg); saveErr != nil {
		agentLog.Warn("failed to persist error message", logger.Error(saveErr))
	}
}

// titleFallback truncates msg to 40 runes as a best-effort conversation title.
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

// buildTitlePrompt constructs the prompt sent to the LLM for title generation.
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

// buildSSECallback constructs the Eino callbacks.Handler that relays tool events
// via AG-UI and persists tool results to the DB for accurate history replay.
// assistantMsgUID is the pre-generated UID for the parent assistant message,
// used as parentMessageId in TOOL_CALL_START events.
// turnUID is the per-turn identifier (equals runID for now; stored in both
// turn_uid and run_uid columns for future retry/sub-task support).
// toolCallsByIndex/toolCallsIndexOrder/sb are pointers into the Chat() frame so
// that ModelCallbackHandler.OnError can save a partial assistant message on failure.
func (a *agentRuntime) buildSSECallback(
	writer agentapi.SSEWriter,
	convID int64,
	convUID string,
	userID int64,
	saveCtx context.Context,
	assistantMsgUID string,
	turnUID string,
	toolCallsByIndex map[int]*schema.ToolCall,
	toolCallsIndexOrder *[]int,
	sb *strings.Builder,
) callbacks.Handler {
	toolHandler := &cbtemplate.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			// compose.GetToolCallID reads the call_id injected by Eino before the callback fires,
			// replacing the previous sync.Map lookup which had a race condition.
			callID := compose.GetToolCallID(ctx)

			// Persist the tool-call decision as a role=assistant message so that:
			// 1. GetConversationMessages reads sm.ToolCalls correctly (snapshot).
			// 2. assembleMessages/convertMessagesToSchema populate knownCallIDs
			//    correctly, preventing all tool result rows from being dropped as orphans.
			toolCallMsgUID := uuid.GenerateShortUUID()
			assistantToolCallMsg := &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:   callID,
					Type: "function",
					Function: schema.FunctionCall{
						Name:      info.Name,
						Arguments: args,
					},
				}},
			}
			rawTC, _ := json.Marshal(assistantToolCallMsg)
			dbTCMsg := &agentmodel.Message{
				ConversationID:  convID,
				ConversationUID: convUID,
				MessageUID:      toolCallMsgUID,
				Role:            schema.Assistant,
				RawMessage:      json.RawMessage(rawTC),
				TurnUID:         turnUID,
				RunUID:          turnUID,
			}
			dbTCMsg.CreatedBy = userID
			dbTCMsg.UpdatedBy = userID
			if err := a.convSrv.SaveMessage(saveCtx, dbTCMsg); err != nil {
				agentLog.Warn("failed to persist tool-call assistant message",
					logger.String("tool", info.Name), logger.Error(err))
			}

			writer.WriteToolCallStart(callID, info.Name, assistantMsgUID)
			writer.WriteToolCallArgs(callID, args)
			writer.WriteToolCallEnd(callID)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			callID := compose.GetToolCallID(ctx)

			// Build guidance from the tool result: attempt JSON parse for raw_data so
			// the frontend can render the response with a rich component (table, list…).
			// When the result is not valid JSON, guidance stays nil and the client falls
			// back to plain-text display.
			var guidance *agui.Guidance
			if result != "" {
				var rawData json.RawMessage
				if json.Unmarshal([]byte(result), &rawData) == nil {
					guidance = &agui.Guidance{
						RenderType: inferRenderType(info.Name),
						RawData:    rawData,
					}
				}
			}

			toolMsgUID := uuid.GenerateShortUUID()
			writer.WriteToolCallResult(toolMsgUID, callID, info.Name, result, guidance)

			// Persist the tool result so future history replay includes the full
			// assistant→tool→assistant chain.
			toolMsg := &schema.Message{
				Role:       schema.Tool,
				Content:    result,
				ToolCallID: callID,
				ToolName:   info.Name,
			}
			raw, _ := json.Marshal(toolMsg)
			dbMsg := &agentmodel.Message{
				ConversationID:  convID,
				ConversationUID: convUID,
				MessageUID:      toolMsgUID,
				Role:            schema.Tool,
				RawMessage:      json.RawMessage(raw),
				ToolCallID:      callID,
				TurnUID:         turnUID,
				RunUID:          turnUID,
			}
			if guidance != nil {
				dbMsg.Guidance, _ = json.Marshal(guidance)
			}
			dbMsg.CreatedBy = userID
			dbMsg.UpdatedBy = userID
			if err := a.convSrv.SaveMessage(saveCtx, dbMsg); err != nil {
				agentLog.Warn("failed to persist tool result",
					logger.String("tool", info.Name), logger.Error(err))
			} else {
				agentLog.Info("stream: tool result persisted",
					logger.String("tool", info.Name), logger.Int("result_len", len(result)))
			}
			return ctx
		},
	}

	// On model-layer error, save any accumulated partial assistant message so that
	// tool-result rows written by OnEnd have a parent, preventing orphaned tool
	// messages from breaking the next LLM API call.
	modelHandler := &cbtemplate.ModelCallbackHandler{
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if len(toolCallsByIndex) > 0 || sb.Len() > 0 {
				a.savePartialAssistantMessage(saveCtx, convID, convUID, userID,
					assistantMsgUID, sb.String(), turnUID, toolCallsByIndex, *toolCallsIndexOrder)
			}
			return ctx
		},
	}
	return react.BuildAgentCallback(modelHandler, toolHandler)
}

// inferRenderType maps a tool name to the suggested frontend render component.
// This drives the guidance.render_type field so the client can pick the right
// rich renderer without inspecting the raw_data structure.
func inferRenderType(toolName string) string {
	switch toolName {
	case "linsight_datasource_query":
		return "table"
	case "linsight_list_datasources", "linsight_search_dashboards":
		return "list"
	default:
		return "text"
	}
}
