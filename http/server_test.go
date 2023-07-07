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

package http

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/config"
)

func TestHTTPServce(t *testing.T) {
	defer func() {
		listenFn = net.Listen
	}()
	s := NewServer(&config.HTTP{Port: 19200})
	assert.NotNil(t, s.GetEngine())
	listenFn = func(_, _ string) (net.Listener, error) {
		return nil, fmt.Errorf("err")
	}
	assert.Error(t, s.Run())

	listenFn = func(_, _ string) (net.Listener, error) {
		return nil, nil
	}
	assert.Panics(t, func() {
		_ = s.Run()
		assert.True(t, false)
	})
}
