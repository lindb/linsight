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

package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitFrontmatter(t *testing.T) {
	t.Run("full block", func(t *testing.T) {
		fm, body := SplitFrontmatter("---\ndescription: hi\n---\nbody text")
		assert.Equal(t, "description: hi", fm)
		assert.Equal(t, "body text", body)
	})

	t.Run("leading whitespace before fence", func(t *testing.T) {
		fm, body := SplitFrontmatter("\n  \n---\nk: v\n---\nb")
		assert.Equal(t, "k: v", fm)
		assert.Equal(t, "b", body)
	})

	t.Run("no frontmatter returns whole content as body", func(t *testing.T) {
		fm, body := SplitFrontmatter("just markdown, no fence")
		assert.Empty(t, fm)
		assert.Equal(t, "just markdown, no fence", body)
	})

	t.Run("unterminated fence is ordinary markdown", func(t *testing.T) {
		fm, body := SplitFrontmatter("---\nk: v\nno closing delimiter")
		assert.Empty(t, fm)
		assert.Equal(t, "---\nk: v\nno closing delimiter", body)
	})

	t.Run("multi-line frontmatter and body", func(t *testing.T) {
		fm, body := SplitFrontmatter("---\na: 1\nb: 2\n---\nline1\nline2")
		assert.Equal(t, "a: 1\nb: 2", fm)
		assert.Equal(t, "line1\nline2", body)
	})

	t.Run("crlf newlines normalised", func(t *testing.T) {
		fm, body := SplitFrontmatter("---\r\nk: v\r\n---\r\nbody")
		assert.Equal(t, "k: v", fm)
		assert.Equal(t, "body", body)
	})
}

func TestParseFrontmatter(t *testing.T) {
	type TestMeta struct {
		Title string `yaml:"title"`
		Order int    `yaml:"order"`
	}

	t.Run("with frontmatter", func(t *testing.T) {
		meta, body, err := ParseFrontmatter[TestMeta]("---\ntitle: Test\norder: 1\n---\nContent here")
		assert.NoError(t, err)
		assert.Equal(t, "Test", meta.Title)
		assert.Equal(t, 1, meta.Order)
		assert.Equal(t, "Content here", body)
	})

	t.Run("no frontmatter", func(t *testing.T) {
		meta, body, err := ParseFrontmatter[TestMeta]("Just content, no frontmatter")
		assert.NoError(t, err)
		assert.Equal(t, TestMeta{}, meta)
		assert.Equal(t, "Just content, no frontmatter", body)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		_, _, err := ParseFrontmatter[TestMeta]("---\ntitle: Test\ninvalid: [unclosed\n---\nContent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse frontmatter")
	})

	t.Run("partial fields", func(t *testing.T) {
		meta, body, err := ParseFrontmatter[TestMeta]("---\ntitle: Only Title\n---\nBody")
		assert.NoError(t, err)
		assert.Equal(t, "Only Title", meta.Title)
		assert.Equal(t, 0, meta.Order)
		assert.Equal(t, "Body", body)
	})
}
