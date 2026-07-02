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
	"io"

	"github.com/cloudwego/eino/schema"
)

// fullResponseStreamToolCallChecker reads all streaming chunks to detect tool calls.
// Unlike the default firstChunkStreamToolCallChecker, this handles LLMs (e.g. DeepSeek)
// that emit text content before tool_calls in the same streaming response — the default
// first-chunk checker would route to END prematurely when it sees text before tool_calls.
//
// The full-read approach is safe to use alongside OnEndWithStreamOutput: the callback
// drains its fan-out copy in real-time (streaming text to SSE), and by the time the
// callback returns the LLM output is fully buffered. This checker then reads from
// that buffer quickly without adding latency.
//
// The handler closes sr before returning as required by the Eino contract.
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
