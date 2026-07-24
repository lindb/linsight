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
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SplitFrontmatter extracts a leading "---\n...\n---" YAML frontmatter block from a
// Markdown document, returning (frontmatter, body). It is shared by the skill, rule, and
// specialist backends, which each YAML-unmarshal the returned frontmatter into their own
// metadata struct while using the body as content.
//
// When content does not begin with a frontmatter delimiter, or the opening "---" has no
// matching closing delimiter, frontmatter is empty and body is the WHOLE original content
// (a leading "---" is then treated as an ordinary Markdown thematic break, not a malformed
// block). Whitespace before the opening delimiter is ignored.
func SplitFrontmatter(content string) (frontmatter, body string) {
	const delimiter = "---"

	trimmed := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmed, delimiter) {
		return "", content
	}

	lines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delimiter {
		return "", content
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delimiter {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	// No closing fence: treat the leading "---" as ordinary Markdown.
	return "", content
}

// ParseFrontmatter splits frontmatter from content, unmarshals it into T, and returns
// (meta, body, err). When no frontmatter is present, meta is zero-valued and err is nil.
func ParseFrontmatter[T any](content string) (meta T, body string, err error) {
	fm, body := SplitFrontmatter(content)
	if fm == "" {
		return meta, body, nil
	}
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return meta, body, fmt.Errorf("parse frontmatter: %w", err)
	}
	return meta, body, nil
}
