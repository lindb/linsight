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

package telemetry

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/pkg/util"
)

// buildLogHandler constructs the Eino callback handler that logs detailed
// per-step content for every component invocation (model input/output, tool
// call/result, and errors). This handler is registered alongside the attribute
// handler so both fire on every Eino node.
func buildLogHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(logOnStart).
		OnEndFn(logOnEnd).
		OnErrorFn(logOnError).
		Build()
}

// nodeLabel returns info.Name if set; otherwise falls back to string(info.Component)
// for inner components not registered as named graph nodes (e.g. the OpenAI model
// inside DynamicModelProxy).
func nodeLabel(info *callbacks.RunInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return string(info.Component)
}

// logOnStart fires when an Eino component begins processing.
// For model components it logs the message history summary and last user message.
// For tool components it logs the tool name and input arguments.
func logOnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil || u.User == nil {
		return ctx
	}

	convID := GetConversationID(ctx)
	orgID := u.Org.ID
	userID := u.User.ID

	// Tool component: type-assert directly to *tool.CallbackInput.
	if toolInput, ok := input.(*tool.CallbackInput); ok {
		args := ""
		if toolInput != nil {
			args = truncate(toolInput.ArgumentsInJSON, 200)
		}
		telLog.Info("agent tool call",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Int64("org.id", orgID),
			logger.Int64("user.id", userID),
			logger.Int64("conv.id", convID),
			logger.String("args", args),
		)
		return ctx
	}

	// Model component: use Eino's helper to extract the typed callback input.
	if modelInput := eino_model.ConvCallbackInput(input); modelInput != nil {
		msgs := modelInput.Messages
		lastMsg := lastUserMessage(msgs)
		telLog.Info("agent model input",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Int64("org.id", orgID),
			logger.Int64("user.id", userID),
			logger.Int64("conv.id", convID),
			logger.Int("messages_count", len(msgs)),
			logger.String("last_user_msg", truncate(lastMsg, 300)),
		)
	}
	return ctx
}

// logOnEnd fires when an Eino component finishes processing.
// For tool components it logs the result string.
// For model components it logs content, tool_calls list, and token usage.
func logOnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil || u.User == nil {
		return ctx
	}

	convID := GetConversationID(ctx)
	orgID := u.Org.ID
	userID := u.User.ID

	// Tool component: type-assert directly to *tool.CallbackOutput.
	if toolOutput, ok := output.(*tool.CallbackOutput); ok {
		result := ""
		if toolOutput != nil {
			result = truncate(toolOutput.Response, 300)
		}
		telLog.Info("agent tool result",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Int64("org.id", orgID),
			logger.Int64("user.id", userID),
			logger.Int64("conv.id", convID),
			logger.String("result", result),
		)
		return ctx
	}

	// Model component: use Eino's helper to extract the typed callback output.
	modelOutput := eino_model.ConvCallbackOutput(output)
	if modelOutput == nil {
		return ctx
	}
	msg := modelOutput.Message
	if msg == nil {
		return ctx
	}

	// Build tool_calls summary: comma-joined function names, or empty string.
	toolCallsStr := ""
	if len(msg.ToolCalls) > 0 {
		names := make([]string, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			names = append(names, tc.Function.Name)
		}
		toolCallsStr = strings.Join(names, ",")
	}

	// Log with token usage when available, without otherwise.
	if modelOutput.TokenUsage != nil {
		telLog.Info("agent model output",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Int64("org.id", orgID),
			logger.Int64("user.id", userID),
			logger.Int64("conv.id", convID),
			logger.String("content", truncate(msg.Content, 300)),
			logger.String("tool_calls", toolCallsStr),
			logger.Int("prompt_tokens", modelOutput.TokenUsage.PromptTokens),
			logger.Int("completion_tokens", modelOutput.TokenUsage.CompletionTokens),
			logger.Int("total_tokens", modelOutput.TokenUsage.TotalTokens),
		)
	} else {
		telLog.Info("agent model output",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Int64("org.id", orgID),
			logger.Int64("user.id", userID),
			logger.Int64("conv.id", convID),
			logger.String("content", truncate(msg.Content, 300)),
			logger.String("tool_calls", toolCallsStr),
		)
	}
	return ctx
}

// logOnError fires when an Eino component returns an error.
func logOnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil || u.User == nil {
		telLog.Error("agent component error",
			logger.String("node", nodeLabel(info)),
			logger.String("type", info.Type),
			logger.String("component", string(info.Component)),
			logger.Error(err),
		)
		return ctx
	}

	convID := GetConversationID(ctx)
	telLog.Error("agent component error",
		logger.String("node", nodeLabel(info)),
		logger.String("type", info.Type),
		logger.String("component", string(info.Component)),
		logger.Int64("org.id", u.Org.ID),
		logger.Int64("user.id", u.User.ID),
		logger.Int64("conv.id", convID),
		logger.Error(err),
	)
	return ctx
}

// lastUserMessage returns the content of the last user-role message in msgs,
// or the last message's content if no user message is found.
// Returns an empty string when msgs is empty.
func lastUserMessage(msgs []*schema.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == schema.User {
			return msgs[i].Content
		}
	}
	if len(msgs) > 0 {
		return msgs[len(msgs)-1].Content
	}
	return ""
}

// truncate returns at most maxRunes runes from s, appending "…" when truncated.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}
