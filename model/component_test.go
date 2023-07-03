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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Components_ToTree(t *testing.T) {
	cmps := Components{
		{UID: "1", Label: "node1", ParentUID: ""},
		{UID: "2", Label: "node2", ParentUID: "1"},
		{UID: "3", Label: "node3", ParentUID: "1"},
		{UID: "4", Label: "node4", ParentUID: "2"},
		{UID: "5", Label: "node5", ParentUID: "2"},
		{UID: "6", Label: "node6", ParentUID: "3"},
		{UID: "7", Label: "node7", ParentUID: "3"},
	}

	tree := cmps.ToTree()
	assert.Len(t, tree, 1)
	assert.Equal(t, "node1", tree[0].Label)
	assert.Len(t, tree[0].Children, 2)
	assert.Equal(t, "node2", tree[0].Children[0].Label)
	assert.Equal(t, "node3", tree[0].Children[1].Label)
	assert.Len(t, tree[0].Children[0].Children, 2)
	assert.Equal(t, "node4", tree[0].Children[0].Children[0].Label)
	assert.Equal(t, "node5", tree[0].Children[0].Children[1].Label)

	assert.Len(t, tree[0].Children[1].Children, 2)
	assert.Equal(t, "node6", tree[0].Children[1].Children[0].Label)
	assert.Equal(t, "node7", tree[0].Children[1].Children[1].Label)
}
