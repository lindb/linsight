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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// makeToolCalls builds n dummy tool calls; only the count matters for dispatch tests.
func makeToolCalls(n int) []schema.ToolCall {
	tcs := make([]schema.ToolCall, n)
	for i := range tcs {
		tcs[i] = schema.ToolCall{ID: fmt.Sprintf("call-%d", i)}
	}
	return tcs
}

func TestDispatchToolCalls_PreservesOrder(t *testing.T) {
	toolCalls := makeToolCalls(5)
	for _, maxPar := range []int{0, 1, 3, 10} {
		t.Run(fmt.Sprintf("maxParallel=%d", maxPar), func(t *testing.T) {
			results := dispatchToolCalls(toolCalls, maxPar, func(i int) *schema.Message {
				return &schema.Message{Role: schema.Tool, Content: fmt.Sprintf("r%d", i), ToolCallID: toolCalls[i].ID}
			})
			assert.Len(t, results, 5)
			// Results must be in the original tool-call order regardless of completion order.
			for i := range results {
				assert.Equal(t, fmt.Sprintf("r%d", i), results[i].Content)
				assert.Equal(t, toolCalls[i].ID, results[i].ToolCallID)
			}
		})
	}
}

func TestDispatchToolCalls_Empty(t *testing.T) {
	assert.Empty(t, dispatchToolCalls(nil, 4, func(int) *schema.Message { return nil }))
}

func TestDispatchToolCalls_RunsConcurrently(t *testing.T) {
	const n = 8
	var inFlight, maxInFlight int32
	var mu sync.Mutex
	block := make(chan struct{})

	var started sync.WaitGroup
	started.Add(n)

	done := make(chan []*schema.Message, 1)
	go func() {
		done <- dispatchToolCalls(makeToolCalls(n), n, func(i int) *schema.Message {
			cur := atomic.AddInt32(&inFlight, 1)
			mu.Lock()
			if cur > maxInFlight {
				maxInFlight = cur
			}
			mu.Unlock()
			started.Done()
			<-block // hold the worker so overlap is observable
			atomic.AddInt32(&inFlight, -1)
			return &schema.Message{Content: fmt.Sprintf("r%d", i)}
		})
	}()

	started.Wait() // all workers reached the barrier => they ran concurrently
	close(block)   // release them
	results := <-done
	assert.Len(t, results, n)
	assert.Equal(t, int32(n), maxInFlight, "all workers should overlap when maxParallel=n")
}

func TestDispatchToolCalls_BoundsConcurrency(t *testing.T) {
	const n, limit = 10, 3
	var inFlight, maxInFlight int32

	results := dispatchToolCalls(makeToolCalls(n), limit, func(i int) *schema.Message {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			old := atomic.LoadInt32(&maxInFlight)
			if cur <= old || atomic.CompareAndSwapInt32(&maxInFlight, old, cur) {
				break
			}
		}
		// tiny spin to encourage overlap without a real sleep
		for j := 0; j < 1000; j++ {
			_ = j
		}
		atomic.AddInt32(&inFlight, -1)
		return &schema.Message{Content: fmt.Sprintf("r%d", i)}
	})

	assert.Len(t, results, n)
	assert.LessOrEqual(t, maxInFlight, int32(limit), "concurrency must not exceed maxParallel")
}
