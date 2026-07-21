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

package subagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	aievent "github.com/lindb/linsight/ai/components/event"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/pkg/util"
)

// stepTitleTimeout caps the lightweight LLM call that names a step, so a slow or stuck
// model never delays closing the step beyond a fixed budget (the caller falls back to
// the "Step N" placeholder on timeout).
const stepTitleTimeout = 8 * time.Second

// Generator is the minimal non-streaming model surface needed to name a step.
// *modelcomp.DynamicModelProxy satisfies it (it exposes both Generate and Stream).
type Generator interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
}

// BuildStepTitlePrompt composes the prompt for naming one reasoning step from its produced
// text and tool calls. It returns "" when the step produced neither text nor tool calls,
// signalling the caller to skip title generation and keep the "Step N" placeholder.
func BuildStepTitlePrompt(content string, toolCalls []schema.ToolCall) string {
	const (
		maxContentRunes = 800
		maxArgsRunes    = 200
	)

	var b strings.Builder
	if trimmed := strings.TrimSpace(content); trimmed != "" {
		b.WriteString("Assistant output:\n")
		b.WriteString(truncateRunes(trimmed, maxContentRunes))
		b.WriteString("\n\n")
	}

	// Collect named tool calls first; only emit the section if at least one has a name, so a
	// step with only unnamed/malformed tool calls does not yield a contentless prompt.
	var toolLines strings.Builder
	for _, tc := range toolCalls {
		name := tc.Function.Name
		if name == "" {
			continue
		}
		if args := strings.TrimSpace(tc.Function.Arguments); args != "" {
			toolLines.WriteString(fmt.Sprintf("- %s(%s)\n", name, truncateRunes(args, maxArgsRunes)))
		} else {
			toolLines.WriteString(fmt.Sprintf("- %s\n", name))
		}
	}
	if toolLines.Len() > 0 {
		b.WriteString("Tool calls made in this step:\n")
		b.WriteString(toolLines.String())
		b.WriteString("\n")
	}

	stepInfo := strings.TrimSpace(b.String())
	if stepInfo == "" {
		return ""
	}
	return "You name one step of an AI agent's work. Below is what the agent produced in this step " +
		"(its text and/or the tools it called).\n" +
		"Reply with ONLY a concise title of at most 6 words describing what this step does. " +
		"No quotes, no trailing punctuation, no extra text. " +
		"Write the title in the SAME language as the content below.\n\n" +
		stepInfo
}

// GenerateStepTitle runs one non-streaming model call to name a step. It returns "" (and
// logs a warning on error) so the caller can fall back to the "Step N" placeholder; the
// title is trimmed of surrounding quotes/whitespace and capped at 60 runes.
func GenerateStepTitle(ctx context.Context, gen Generator, content string, toolCalls []schema.ToolCall) string {
	prompt := BuildStepTitlePrompt(content, toolCalls)
	if prompt == "" {
		return ""
	}
	msg, err := gen.Generate(ctx, []*schema.Message{{Role: schema.User, Content: prompt}})
	if err != nil {
		subLog.Warn("step title generation failed, keeping placeholder", logger.Error(err))
		return ""
	}
	return sanitizeStepTitle(msg.Content)
}

// StartStepTitle spawns a goroutine that generates the step title concurrently (overlapping
// tool execution) and returns a buffered channel delivering exactly one result ("" on skip,
// failure, or timeout). The step's OnFinish reads this channel to name the step.
//
// authCtx must carry the signed user (for per-org model resolution); generation runs on an
// INDEPENDENT background context bounded by stepTitleTimeout, so it is unaffected by
// authCtx's (or the graph node's) cancellation once the step advances. The channel is
// buffered so the goroutine never blocks even if OnFinish is skipped.
func StartStepTitle(authCtx context.Context, gen Generator, content string, toolCalls []schema.ToolCall) <-chan string {
	ch := make(chan string, 1)
	user := util.GetUser(authCtx)
	go func() {
		tctx, cancel := context.WithTimeout(
			context.WithValue(context.Background(), constant.LinSightSignedKey, user),
			stepTitleTimeout,
		)
		defer cancel()
		ch <- GenerateStepTitle(tctx, gen, content, toolCalls)
	}()
	return ch
}

// sanitizeStepTitle trims whitespace and wrapping quotes and caps the title length.
func sanitizeStepTitle(s string) string {
	const maxRunes = 60
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'`")
	s = strings.TrimSpace(s)
	if runes := []rune(s); len(runes) > maxRunes {
		s = string(runes[:maxRunes])
	}
	return s
}

// truncateRunes returns s truncated to at most n runes.
func truncateRunes(s string, n int) string {
	if runes := []rune(s); len(runes) > n {
		return string(runes[:n])
	}
	return s
}

// DrainModelStream consumes one LLM stream, forwarding reasoning/text deltas to the
// event objects and merging streaming tool_call fragments (OpenAI protocol) in index
// order. It ALWAYS closes the reasoning and text SSE segments (OnEnd, via defer) — on both
// EOF and a mid-stream receive error — so a failed stream never leaves a perpetually-open
// "streaming" bubble or an unpersisted partial segment. OnEnd is a no-op for a segment that
// never started, so closing is safe regardless of how far the stream got.
//
// It is shared verbatim by the host execution node and the sub-agent runner so both speak
// the identical streaming protocol.
func DrainModelStream(
	ctx context.Context,
	reader *schema.StreamReader[*schema.Message],
	asst *aievent.AssistantTextEvent,
	reasoning *aievent.ReasoningEvent,
) (content string, toolCalls []schema.ToolCall, err error) {
	// Close reasoning + text SSE frames for this model call on every exit path.
	defer func() {
		reasoning.OnEnd(ctx)
		asst.OnEnd(ctx)
	}()

	// toolCallsByIndex accumulates partial ToolCall fragments keyed by the OpenAI streaming
	// index; toolCallsIndexOrder preserves insertion order for deterministic merging.
	toolCallsByIndex := make(map[int]*schema.ToolCall)
	var toolCallsIndexOrder []int
	var contentSB strings.Builder

	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return "", nil, fmt.Errorf("subagent stream recv: %w", recvErr)
		}
		if chunk == nil {
			continue
		}

		if chunk.ReasoningContent != "" {
			reasoning.OnContent(chunk.ReasoningContent)
		}
		if chunk.Content != "" {
			asst.OnContent(chunk.Content)
			contentSB.WriteString(chunk.Content)
		}

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

	if len(toolCallsByIndex) > 0 {
		toolCalls = make([]schema.ToolCall, 0, len(toolCallsByIndex))
		for _, idx := range toolCallsIndexOrder {
			toolCalls = append(toolCalls, *toolCallsByIndex[idx])
		}
	}
	return contentSB.String(), toolCalls, nil
}
