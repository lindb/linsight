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

package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/ai/telemetry"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// chatRun holds all mutable state for one conversation turn.
// A new chatRun is created per Chat() call; shared singleton fields are accessed
// through the agent pointer to keep the boundary explicit.
type chatRun struct {
	agent       *agentRuntime
	writer      agentapi.SSEWriter
	req         *agentapi.ChatRequest
	reqCtx      context.Context // original request context; carries the auth user key
	initConvUID string          // "" means create a new conversation

	// resolved by initConversation
	orgID  int64
	userID int64
	conv   *agentmodel.Conversation
	isNew  bool

	// set at the very start of run()
	runID string
	start time.Time

	// per-request timeout context (inherits auth from reqCtx)
	timeoutCtx context.Context
	cancel     context.CancelFunc

	// enriched context forwarded to the ReAct graph (conversation + skill telemetry)
	runCtx context.Context

	// tool + SSE callback options built up by resolveTools and run()
	agentOpts []einoagent.AgentOption

	// streaming accumulators; written by callbacks, read by finalize
	toolCallsByIndex    map[int]*schema.ToolCall // keyed by OpenAI stream Index field
	toolCallsIndexOrder []int                    // insertion order for deterministic merging
	sb, rsb             strings.Builder          // text content and reasoning content
	textStarted         bool                     // true once WriteTextMessageStart has fired
	partialSaved        bool                     // guards against double-saving partial content

	// pre-generated UIDs known before streaming begins
	assistantMsgUID string
	userMsgUID      string
}

// newChatRun allocates a chatRun for one conversation turn.
func newChatRun(
	a *agentRuntime,
	ctx context.Context,
	req *agentapi.ChatRequest,
	writer agentapi.SSEWriter,
	convUID string,
) *chatRun {
	return &chatRun{
		agent:            a,
		writer:           writer,
		req:              req,
		reqCtx:           ctx,
		initConvUID:      convUID,
		toolCallsByIndex: make(map[int]*schema.ToolCall),
	}
}

// run executes the full conversation turn in sequential phases and delivers AG-UI
// events over rt.writer. The named return allows defers to set retErr directly.
//
// Defer registration order (LIFO — last registered executes first):
//  1. panic recovery  (registered first → executes last)
//  2. cancel
//  3. semaphore release
//  4. fallback title  (if new conversation)
//  5. persistErrorMessage
//  6. persistAssistantMessage on error
//  7. WriteTextMessageEnd on error
//  8. reader.Close  (registered inside executeStream)
func (rt *chatRun) run() (retErr error) {
	rt.runID = uuid.GenerateShortUUID()
	rt.start = time.Now()

	// Panic recovery — registered first (LIFO) so it runs last, after all cleanup
	// defers have finished.
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("agent panic: %v", r)
			agentLog.Error("agent panic recovered", logger.String("panic", fmt.Sprintf("%v", r)))
			retErr = panicErr
			if rt.isNew && rt.conv != nil {
				rt.agent.persistFallbackTitle(rt.reqCtx, rt.conv.ID, rt.req.Content)
			}
		}
	}()

	rt.timeoutCtx, rt.cancel = context.WithTimeout(rt.reqCtx, rt.agent.cfg.RequestTimeout.Duration())
	defer rt.cancel()

	user := util.GetUser(rt.reqCtx)
	rt.orgID = user.Org.ID
	rt.userID = user.User.ID

	// Per-org concurrency guard: reject immediately if the semaphore is full.
	sem := rt.agent.getSemaphore(rt.orgID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		return agentapi.ErrTooManyRequests
	}

	if err := rt.initConversation(); err != nil {
		return err
	}
	rt.writer.WriteRunStarted(rt.runID, rt.conv.UID)

	// For new conversations: persist a fallback title when we exit with an error so
	// the sidebar never shows an empty entry.
	if rt.isNew {
		defer func() {
			if retErr != nil {
				rt.agent.persistFallbackTitle(rt.reqCtx, rt.conv.ID, rt.req.Content)
			}
		}()
	}

	// Persist an error message on the way out so failures are visible after page
	// refresh. Guard: only fire once a user message has been persisted (userMsgUID set).
	defer func() {
		if retErr != nil && rt.userMsgUID != "" {
			rt.persistAgentError(rt.reqCtx, rt.runID, retErr)
		}
	}()

	rt.assistantMsgUID = uuid.GenerateShortUUID()

	if err := rt.persistUserMessage(rt.timeoutCtx); err != nil {
		return err
	}

	messages, err := rt.buildContext()
	if err != nil {
		return err
	}

	// resolveTools is non-fatal: it logs and continues if the registry call fails.
	rt.resolveTools()

	// On error: save accumulated text so tool-result rows written by OnEnd have a
	// parent — preventing orphaned tool messages from breaking the next LLM call.
	// Registered before the text-close defer (LIFO), so this runs after stream close.
	defer func() {
		if retErr != nil && (rt.sb.Len() > 0 || rt.rsb.Len() > 0) && !rt.partialSaved {
			rt.partialSaved = true
			rt.persistAssistantContent(rt.reqCtx)
		}
	}()

	// Close the text stream before RUN_ERROR so the frontend never sees an unclosed
	// TEXT_MESSAGE_START when the turn returns an error.
	defer func() {
		if retErr != nil && rt.textStarted {
			rt.writer.WriteTextMessageEnd(rt.assistantMsgUID)
		}
	}()

	sseCallback := newSSECallback(rt)
	rt.agentOpts = append(rt.agentOpts, einoagent.WithComposeOptions(compose.WithCallbacks(sseCallback)))

	if err := rt.executeStream(messages); err != nil {
		return err
	}
	return rt.finalize()
}

// initConversation resolves an existing conversation or creates a new one.
func (rt *chatRun) initConversation() error {
	rt.isNew = rt.initConvUID == ""
	var err error
	if rt.isNew {
		rt.conv, err = rt.agent.convSrv.CreateConversation(rt.timeoutCtx)
	} else {
		rt.conv, err = rt.agent.convSrv.GetConversation(rt.timeoutCtx, rt.initConvUID)
	}
	if err != nil {
		agentLog.Error("failed to initialize conversation", logger.Error(err))
		return fmt.Errorf("get/create conversation: %w", err)
	}
	return nil
}

// buildContext loads conversation history and assembles the LLM message list.
func (rt *chatRun) buildContext() ([]*schema.Message, error) {
	messages, err := rt.agent.buildLLMMessages(rt.timeoutCtx, rt.conv, rt.req.Content)
	if err != nil {
		agentLog.Error("failed to build LLM messages", logger.Error(err))
		return nil, fmt.Errorf("build messages: %w", err)
	}
	return messages, nil
}

// resolveTools loads the governance-filtered tool set, wires it into agentOpts, and
// sets up the enriched run context with conversation and skill-metadata telemetry.
// Non-fatal: logs a warning and continues with an empty tool set on failure.
func (rt *chatRun) resolveTools() {
	orgTools, skillMeta, err := rt.agent.skillRegistry.LoadConversationTools(rt.timeoutCtx, rt.orgID, rt.conv.ID)
	if err != nil {
		agentLog.Warn("failed to load conversation tools, proceeding without tools", logger.Error(err))
	}

	agentLog.Info("chat session start",
		logger.Int64("org.id", rt.orgID),
		logger.Int64("user.id", rt.userID),
		logger.String("conv.uid", rt.conv.UID),
		logger.Any("is_new", rt.isNew),
		logger.Int("tools_count", len(orgTools)),
	)

	metaMap := buildSkillMetaMap(skillMeta)
	rt.runCtx = telemetry.WithConversationID(rt.timeoutCtx, rt.conv.ID)
	rt.runCtx = telemetry.WithSkillMetaMap(rt.runCtx, metaMap)

	if len(orgTools) == 0 {
		return
	}
	toolOpts, toolErr := react.WithTools(rt.runCtx, orgTools...)
	if toolErr != nil {
		agentLog.Warn("react.WithTools failed, proceeding without tools", logger.Error(toolErr))
		return
	}
	rt.agentOpts = append(rt.agentOpts, toolOpts...)
}

// executeStream starts the ReAct graph, reads chunks from the stream, and merges
// tool_call fragments by Index (OpenAI streaming protocol).
// Text content is delivered in real-time via ModelCallbackHandler.OnEndWithStreamOutput;
// this loop only handles tool_call accumulation.
func (rt *chatRun) executeStream(messages []*schema.Message) error {
	reader, err := rt.agent.compiled.Stream(rt.runCtx, messages, rt.agentOpts...)
	if err != nil {
		agentLog.Error("failed to start agent stream", logger.Error(err))
		return fmt.Errorf("agent stream: %w", err)
	}
	defer reader.Close()

	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			agentLog.Error("stream recv error", logger.Error(recvErr))
			return fmt.Errorf("stream recv: %w", recvErr)
		}
		if chunk == nil || chunk.Role != schema.Assistant || len(chunk.ToolCalls) == 0 {
			continue
		}
		// Merge streaming tool_call fragments: each chunk carries a partial Arguments
		// string that must be concatenated in Index order.
		for i := range chunk.ToolCalls {
			tc := &chunk.ToolCalls[i]
			idx := i
			if tc.Index != nil {
				idx = *tc.Index
			}
			if existing, ok := rt.toolCallsByIndex[idx]; ok {
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
				rt.toolCallsByIndex[idx] = &copied
				rt.toolCallsIndexOrder = append(rt.toolCallsIndexOrder, idx)
			}
		}
	}
	return nil
}

// finalize persists the completed assistant message (content + reasoningContent only —
// tool calls are already saved per-invocation by ToolCallbackHandler.OnStart),
// generates a title for new conversations, and emits RUN_FINISHED.
//
// Why tool calls are excluded: saving them here would create a spurious merged
// assistant row alongside the individual OnStart rows, corrupting the
// assistant→tool→assistant chain used for multi-turn LLM context.
func (rt *chatRun) finalize() error {
	if rt.sb.Len() > 0 || rt.rsb.Len() > 0 {
		rt.persistAssistantContent(rt.timeoutCtx)
	}

	var title string
	if rt.isNew {
		title = rt.agent.generateTitle(rt.reqCtx, rt.conv.ID, rt.req.Content)
	}

	agentLog.Info("chat session complete",
		logger.String("conv.uid", rt.conv.UID),
		logger.Int("response_len", rt.sb.Len()),
		logger.String("elapsed", time.Since(rt.start).String()),
	)

	rt.writer.WriteRunFinished(rt.runID, rt.conv.UID, title)
	return nil
}

// buildSkillMetaMap converts a SkillMeta slice into a name → ControlLevel lookup map
// for telemetry annotation on the run context.
func buildSkillMetaMap(meta []skillscomp.SkillMeta) map[string]agentmodel.SkillControlLevel {
	if len(meta) == 0 {
		return nil
	}
	m := make(map[string]agentmodel.SkillControlLevel, len(meta))
	for _, sm := range meta {
		m[sm.Name] = sm.ControlLevel
	}
	return m
}
