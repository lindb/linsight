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

package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvContext_Seq(t *testing.T) {
	c := &ConvContext{}
	assert.Equal(t, 1, c.NextSeq())
	assert.Equal(t, 2, c.NextSeq())

	// SetSeq primes the counter so the next NextSeq continues from n+1 — the resume path
	// uses this to continue the interrupted run's sequence instead of restarting from 1.
	c.SetSeq(10)
	assert.Equal(t, 11, c.NextSeq())
	assert.Equal(t, 12, c.NextSeq())
}
