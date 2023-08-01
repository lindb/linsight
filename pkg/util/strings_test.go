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

package util

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetRandomString(t *testing.T) {
	defer func() {
		randFn = rand.Read
	}()

	randFn = func(b []byte) (n int, err error) {
		return 0, fmt.Errorf("err")
	}
	str, err := GetRandomString(10)
	assert.Error(t, err)
	assert.Empty(t, str)

	randFn = rand.Read

	str, err = GetRandomString(10)
	assert.NoError(t, err)
	assert.Len(t, str, 10)

	str, err = GetRandomString(10, 1, 2, 3)
	assert.NoError(t, err)
	assert.Len(t, str, 10)
}

func Test_RandomHex(t *testing.T) {
	defer func() {
		randFn = rand.Read
	}()

	randFn = func(b []byte) (n int, err error) {
		return 0, fmt.Errorf("err")
	}
	str, err := RandomHex(10)
	assert.Error(t, err)
	assert.Empty(t, str)

	randFn = rand.Read
	str, err = RandomHex(10)
	assert.NoError(t, err)
	assert.NotEmpty(t, str)
}

func Test_RemoveDuplicates(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, RemoveDuplicates([]string{"a", "b"}))
	assert.Equal(t, []string{"a", "b"}, RemoveDuplicates([]string{"a", "b", "b"}))
}
