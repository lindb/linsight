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

// Package telemetry registers Eino callback handlers that capture linsight-specific
// context (org.id, user.id, conversation.id, tool.control_level) on every component
// invocation.
//
// Context injection: before calling reactAgent.Stream, the service layer must inject
// telemetry attributes into the request context using the helper functions exported by
// this package (WithConversationID, WithSkillMetaMap). The global Eino callback handler
// registered by Register() reads those values from the context on each node invocation.
//
// When the project integrates OpenTelemetry tracing (go.opentelemetry.io/otel), replace
// the log.Debug calls in onStart with span.SetAttributes:
//
//	span := trace.SpanFromContext(ctx)
//	span.SetAttributes(
//	    attribute.Int64("org.id",          orgID),
//	    attribute.Int64("user.id",         userID),
//	    attribute.Int64("conversation.id", convID),
//	    attribute.String("tool.control_level", string(level)),
//	)
package telemetry

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/lindb/common/pkg/logger"

	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/util"
)

var telLog = logger.GetLogger("agent", "telemetry")

// contextKey is an unexported type for context keys owned by this package.
// Using a named type prevents collisions with keys from other packages.
type contextKey int

const (
	conversationIDKey contextKey = iota
	skillMetaMapKey
)

// ── Context helpers ───────────────────────────────────────────────────────────

// WithConversationID stores the active conversation ID in ctx so that the telemetry
// callback can include it in every log/span emitted during the agent run.
func WithConversationID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, conversationIDKey, id)
}

// GetConversationID retrieves the conversation ID stored by WithConversationID.
// Returns 0 if the value is not present.
func GetConversationID(ctx context.Context) int64 {
	v, _ := ctx.Value(conversationIDKey).(int64)
	return v
}

// WithSkillMetaMap stores a tool-name → SkillControlLevel map in ctx.
// The telemetry callback uses it to annotate each tool invocation with its
// governance level ("global" or "conversation").
func WithSkillMetaMap(ctx context.Context, m map[string]agentmodel.SkillControlLevel) context.Context {
	return context.WithValue(ctx, skillMetaMapKey, m)
}

// GetSkillControlLevel looks up the control level for a tool from the map stored by
// WithSkillMetaMap. Returns an empty string if the map is absent or the tool is unknown.
func GetSkillControlLevel(ctx context.Context, toolName string) agentmodel.SkillControlLevel {
	m, _ := ctx.Value(skillMetaMapKey).(map[string]agentmodel.SkillControlLevel)
	return m[toolName]
}

// ── Global handler registration ───────────────────────────────────────────────

// Register appends the linsight attribute handler to the Eino global callback
// chain. It must be called once during program initialization, before any
// graph executions begin. Calling it concurrently with running graphs causes
// data races (Eino's AppendGlobalHandlers is not thread-safe).
func Register() {
	callbacks.AppendGlobalHandlers(buildAttrHandler())
	callbacks.AppendGlobalHandlers(buildLogHandler())
}

// buildAttrHandler constructs the Eino callback handler that annotates every
// component invocation with the request's org, user, conversation, and skill metadata.
func buildAttrHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(onStart).
		Build()
}

// onStart fires at the start of each Eino component invocation.
// It reads telemetry attributes from ctx and logs them alongside the component's
// node name so that observability tools can correlate LLM calls to their
// originating tenant, user, and conversation.
//
// For tool invocations, the governance level (tool.control_level) is also logged
// to provide an audit trail of which skills were used and at what permission tier.
func onStart(ctx context.Context, info *callbacks.RunInfo, _ callbacks.CallbackInput) context.Context {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil || u.User == nil {
		return ctx
	}

	convID := GetConversationID(ctx)
	controlLevel := GetSkillControlLevel(ctx, info.Name)

	if controlLevel != "" {
		// info.Name is a known tool name — emit a tool-specific log entry.
		telLog.Debug("agent tool invocation",
			logger.String("node", info.Name),
			logger.String("type", info.Type),
			logger.Int64("org.id", u.Org.ID),
			logger.Int64("user.id", u.User.ID),
			logger.Int64("conversation.id", convID),
			logger.String("tool.name", info.Name),
			logger.String("tool.control_level", string(controlLevel)),
		)
	} else {
		telLog.Debug("agent component start",
			logger.String("node", info.Name),
			logger.String("type", info.Type),
			logger.Int64("org.id", u.Org.ID),
			logger.Int64("user.id", u.User.ID),
			logger.Int64("conversation.id", convID),
		)
	}
	return ctx
}
