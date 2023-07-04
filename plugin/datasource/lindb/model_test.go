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

package lindb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpr_String(t *testing.T) {
	assert.Equal(t, "key = 'value'", Expr{Key: "key", Op: Eq, Value: "value"}.String())
	assert.Equal(t, "key = '123'", Expr{Key: "key", Op: Eq, Value: 123}.String())
	assert.Equal(t, "key = value", Expr{Key: "key", Op: Eq, Value: "value", raw: true}.String())
	assert.Equal(t, "key in ( '123','abc' )", Expr{Key: "key", Op: In, Value: []any{"123", "abc"}}.String())
}
