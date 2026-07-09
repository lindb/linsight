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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
	aievent "github.com/lindb/linsight/ai/components/event"
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

	// enriched context forwarded to the graph (conversation + skill telemetry)
	runCtx context.Context

	// event lifecycle objects; initialized in run() after conversation is resolved
	convCtx        *aievent.ConvContext
	assistantEvt   *aievent.AssistantTextEvent
	currentStepID  string             // updated at the start of each model-call iteration
	currentStepEvt *aievent.StepEvent // step lifecycle for the current model-call iteration

	// orgTools holds the resolved tool set; used by the graph Lambda nodes via context.
	orgTools []tool.BaseTool

	// pre-generated UIDs known before streaming begins
	assistantMsgUID string
	userMsgUID      string

	// interrupt / resume state.
	// isResume is true when req.Resume != nil; set before run() enters the graph.
	isResume bool
	// resumeConsumed prevents the LLM-skip optimisation from firing on subsequent
	// thinking iterations (e.g. after a rejection routes back to thinking).
	resumeConsumed bool
	// resumeHandled is set by processResumeDecision to prevent re-entry into Case B.
	resumeHandled bool
	// interruptHandled is set when processInterrupt issues an interrupt; tells
	// finalize() to emit WriteRunInterrupt instead of WriteRunFinished.
	interruptHandled bool
	// pendingInterrupt holds the InterruptRecord loaded from DB on a resume run.
	pendingInterrupt *agentmodel.InterruptRecord
	// interruptPayload is built by processInterrupt and consumed by finalize().
	interruptPayload *agui.InterruptPayload
	// pendingToolCallsToExecute carries the approved (and possibly arg-edited)
	// tool calls from processResumeDecision to createToolsNode.
	pendingToolCallsToExecute []schema.ToolCall
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
		agent:       a,
		writer:      writer,
		req:         req,
		reqCtx:      ctx,
		initConvUID: convUID,
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
//  8. reader.Close  (registered inside executeGraph)
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

	// For resume runs, load and validate the interrupt record before emitting
	// RUN_STARTED so the caller gets a clean error on invalid interrupt IDs.
	if rt.req.Resume != nil {
		ir, irErr := rt.agent.convSrv.GetInterruptRecord(rt.timeoutCtx, rt.req.Resume.InterruptID)
		if irErr != nil {
			return fmt.Errorf("load interrupt record: %w", irErr)
		}
		if ir.ConvUID != rt.conv.UID {
			return fmt.Errorf("interrupt %q does not belong to conversation %q",
				rt.req.Resume.InterruptID, rt.conv.UID)
		}
		rt.isResume = true
		rt.pendingInterrupt = ir

		// Reuse the interrupted run's RunID so the resumed events (tool results + the
		// continuation text) are grouped into the SAME run as the original turn. The
		// frontend groups bubbles by runId; a fresh RunID here would render the
		// continuation as a brand-new conversation turn instead of an extension of the
		// interrupted one. Must happen before WriteRunStarted and convCtx creation below.
		if ir.OrigRunID != "" {
			rt.runID = ir.OrigRunID
		}
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
			rt.persistAgentError(rt.reqCtx, retErr)
		}
	}()

	rt.assistantMsgUID = uuid.GenerateShortUUID()

	// Initialize event lifecycle objects; must be before persistUserMessage so
	// the user-message event can use convCtx.NextSeq().
	rt.convCtx = &aievent.ConvContext{
		ConvID:  rt.conv.ID,
		ConvUID: rt.conv.UID,
		UserID:  rt.userID,
		RunID:   rt.runID,
	}
	// Resume reuses the interrupted run's RunID; prime the sequence counter with that run's
	// existing max sequence so resumed events continue its monotonic order instead of
	// colliding from 1. Non-fatal: on error we fall back to 0 (the pre-existing behaviour).
	if rt.isResume {
		maxSeq, seqErr := rt.agent.convSrv.GetMaxEventSequence(rt.timeoutCtx, rt.conv.UID, rt.runID)
		if seqErr != nil {
			agentLog.Warn("resume: failed to load max event sequence, starting from 0",
				logger.String("conv.uid", rt.conv.UID), logger.String("run.id", rt.runID),
				logger.Error(seqErr))
		} else {
			rt.convCtx.SetSeq(maxSeq)
		}
	}
	rt.assistantEvt = aievent.NewAssistantTextEvent(rt.assistantMsgUID, rt.writer, rt.agent.convSrv, rt.convCtx)

	// Resume turns carry no new user content; skip persisting a user message.
	if !rt.isResume {
		if err := rt.persistUserMessage(rt.timeoutCtx); err != nil {
			return err
		}
	}

	messages, err := rt.buildContext()
	if err != nil {
		return err
	}

	// resolveTools is non-fatal: it logs and continues if the registry call fails.
	rt.resolveTools()

	// On error: persist any accumulated assistant text so tool-result rows have a
	// parent — preventing orphaned tool messages from breaking the next LLM call.
	// Registered before the text-close defer (LIFO), so this runs after stream close.
	defer func() {
		if retErr != nil {
			rt.assistantEvt.Persist(rt.reqCtx, rt.currentStepID)
		}
	}()

	// Close any open step on error so the frontend never sees an unclosed STEP_STARTED.
	// Idempotent: no-op if the step was already closed by the normal path.
	defer func() {
		if retErr != nil && rt.currentStepEvt != nil {
			rt.currentStepEvt.OnFinish(rt.reqCtx)
		}
	}()

	// Close the text stream before RUN_ERROR so the frontend never sees an unclosed
	// TEXT_MESSAGE_START when the turn returns an error.
	defer func() {
		if retErr != nil && rt.assistantEvt.IsStarted() {
			rt.assistantEvt.OnEnd(rt.reqCtx)
		}
	}()

	// The graph runs the explicit four-node Lambda pipeline, which also handles
	// AG-UI tool-bound interrupts and resume (human-in-the-loop approval).
	if err := rt.executeGraph(messages); err != nil {
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
// On a resume run it forwards the pending interrupt's tool calls so the history assembler
// can re-append them as the trailing assistant message (the resume graph relies on the last
// message being an assistant-with-tool_calls).
func (rt *chatRun) buildContext() ([]*schema.Message, error) {
	pendingToolCalls := rt.pendingInterruptToolCalls()
	messages, err := rt.agent.buildLLMMessages(rt.timeoutCtx, rt.conv, pendingToolCalls)
	if err != nil {
		agentLog.Error("failed to build LLM messages", logger.Error(err))
		return nil, fmt.Errorf("build messages: %w", err)
	}
	return messages, nil
}

// pendingInterruptToolCalls returns the tool calls of the interrupt being resumed, or nil
// when this is not a resume run. Unmarshal failures are logged and treated as "no pending
// calls" so a corrupt record degrades to the ordinary sanitize path instead of aborting the
// turn (the resume graph will then simply find nothing to execute).
func (rt *chatRun) pendingInterruptToolCalls() []schema.ToolCall {
	if !rt.isResume || rt.pendingInterrupt == nil || len(rt.pendingInterrupt.PendingToolCalls) == 0 {
		return nil
	}
	var tcs []schema.ToolCall
	if err := json.Unmarshal(rt.pendingInterrupt.PendingToolCalls, &tcs); err != nil {
		agentLog.Warn("resume: failed to unmarshal pending tool calls for context",
			logger.String("interrupt.uid", rt.pendingInterrupt.UID), logger.Error(err))
		return nil
	}
	return tcs
}

// resolveTools loads the governance-filtered tool set and sets up the enriched run
// context with conversation and skill-metadata telemetry.
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

	// Store tools for the graph Lambda nodes, which bind them per-request via context.
	rt.orgTools = orgTools
}

// finalize persists the completed assistant message (via AssistantTextEvent.Persist),
// generates a title for new conversations, and emits RUN_FINISHED or RUN_INTERRUPT.
func (rt *chatRun) finalize() error {
	// Close any open step — idempotent; handles the edge case where IsFinished=true
	// is detected alongside tool calls, causing decideBranch to terminate before
	// createToolsNode ran and closed the step.
	if rt.currentStepEvt != nil {
		rt.currentStepEvt.OnFinish(rt.timeoutCtx)
	}

	// Interrupt path: processInterrupt already called assistantEvt.Persist; skip the
	// second call to avoid a duplicate DB row.
	if !rt.interruptHandled {
		rt.assistantEvt.Persist(rt.timeoutCtx, rt.currentStepID)
	}

	var title string
	if rt.isNew {
		title = rt.agent.generateTitle(rt.reqCtx, rt.conv.ID, rt.req.Content)
	}

	agentLog.Info("chat session complete",
		logger.String("conv.uid", rt.conv.UID),
		logger.String("elapsed", time.Since(rt.start).String()),
	)

	if rt.interruptHandled && rt.interruptPayload != nil {
		rt.writer.WriteRunInterrupt(rt.runID, rt.conv.UID, rt.interruptPayload)
	} else {
		rt.writer.WriteRunFinished(rt.runID, rt.conv.UID, title)
	}
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

// enrichRunCtx injects the current *chatRun into the run context so that the
// graph Lambda nodes can access all per-request state (tools, events, writer)
// via a single context key without additional coupling.
func (rt *chatRun) enrichRunCtx() context.Context {
	return context.WithValue(rt.runCtx, ctxChatRun, rt)
}

// executeGraph runs the explicit four-node Lambda graph and drains the output
// stream. All SSE delivery and DB persistence are handled inside the Lambda
// nodes themselves; we only check for stream errors here.
func (rt *chatRun) executeGraph(messages []*schema.Message) error {
	reader, err := rt.agent.graph.Stream(rt.enrichRunCtx(), messages)
	if err != nil {
		return fmt.Errorf("graph stream: %w", err)
	}
	defer reader.Close()

	for {
		_, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return fmt.Errorf("graph recv: %w", recvErr)
		}
	}
	return nil
}
