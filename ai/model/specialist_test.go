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

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateName(t *testing.T) {
	valid := []string{"metrics-analyst", "log_investigator", "a", "Agent1", "x-y_z-9"}
	for _, n := range valid {
		assert.NoError(t, ValidateName(n), "expected %q to be valid", n)
	}

	invalid := []string{"", "1agent", "has space", "bad!", "-leading", "_leading", "with.dot"}
	for _, n := range invalid {
		assert.Error(t, ValidateName(n), "expected %q to be invalid", n)
	}

	// Too long: "handoff_to_" + name must stay within OpenAI's 64-char function-name limit.
	assert.Error(t, ValidateName(strings.Repeat("a", maxSpecialistNameLen+1)))
	assert.NoError(t, ValidateName(strings.Repeat("a", maxSpecialistNameLen)))
}
