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

	"github.com/cloudwego/eino/schema"

	aievent "github.com/lindb/linsight/ai/components/event"
)

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
