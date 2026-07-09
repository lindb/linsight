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
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/pkg/util"
)

// titleGenTimeout caps LLM title generation and fallback DB writes so they never
// block the response path beyond a fixed budget.
const titleGenTimeout = 10 * time.Second

// langAlignInstruction is appended to the system message on every model call to keep
// both reasoning tokens and final text in the same language as the user's input.
const langAlignInstruction = "\n\nIMPORTANT: Always use the SAME language as the user's message for BOTH your reasoning/thinking AND your final response. If the user writes in Chinese, think and respond in Chinese. If the user writes in English, think and respond in English."

// buildLanguageAlignModifier returns a MessageModifier that appends langAlignInstruction
// to the system message before each LLM call, ensuring the model reasons and responds
// in the same language the user wrote in.
// If no system message is present (empty SystemPrompt), a new one is prepended.
func buildLanguageAlignModifier() react.MessageModifier {
	return func(_ context.Context, input []*schema.Message) []*schema.Message {
		for i, msg := range input {
			if msg.Role != schema.System {
				continue
			}
			// Copy to avoid mutating the caller's slice.
			updated := *msg
			updated.Content += langAlignInstruction
			out := make([]*schema.Message, len(input))
			copy(out, input)
			out[i] = &updated
			return out
		}
		// No system message found; prepend a minimal one with just the language instruction.
		out := make([]*schema.Message, 0, len(input)+1)
		out = append(out, &schema.Message{Role: schema.System, Content: strings.TrimPrefix(langAlignInstruction, "\n\n")})
		out = append(out, input...)
		return out
	}
}

// rulesInjectionHeader precedes injected rule text inside the system message so the
// model can distinguish operator-defined rules from the base persona prompt.
const rulesInjectionHeader = "\n\n# Global Rules\nThe following rules are configured by the operator and MUST be strictly followed:\n\n"

// injectRulesIntoSystem returns a NEW message slice with rulesText appended to the
// system message (or a new system message prepended if none exists). It performs a
// copy-on-write so the caller's `messages` slice — which is reused across deep-mode
// graph iterations — is never mutated; otherwise rule blocks would stack up on every
// loop. Mirrors buildLanguageAlignModifier's copy-on-write approach.
func injectRulesIntoSystem(messages []*schema.Message, rulesText string) []*schema.Message {
	if rulesText == "" {
		return messages
	}
	for i, msg := range messages {
		if msg.Role != schema.System {
			continue
		}
		updated := *msg
		updated.Content += rulesInjectionHeader + rulesText
		out := make([]*schema.Message, len(messages))
		copy(out, messages)
		out[i] = &updated
		return out
	}
	// No system message present; prepend one carrying just the rules.
	out := make([]*schema.Message, 0, len(messages)+1)
	out = append(out, &schema.Message{
		Role:    schema.System,
		Content: strings.TrimPrefix(rulesInjectionHeader, "\n\n") + rulesText,
	})
	return append(out, messages...)
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

// persistFallbackTitle writes a truncated user message as the conversation title.
// Called when an error prevents normal title generation so the sidebar never shows an empty entry.
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
