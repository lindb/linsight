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

package integrations

// Manifest represents the structured metadata from manifest.yaml.
type Manifest struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Category    string   `json:"category" yaml:"category"`
	Description string   `json:"description" yaml:"description"`
	Logo        string   `json:"logo,omitempty" yaml:"logo,omitempty"`
	Tags        []string `json:"tags" yaml:"tags"`
	Symptoms    []string `json:"symptoms" yaml:"symptoms"`
	Maintainers []string `json:"maintainers,omitempty" yaml:"maintainers,omitempty"`
	Homepage    string   `json:"homepage,omitempty" yaml:"homepage,omitempty"`
}

// ContentSection represents a section from a markdown file with frontmatter metadata.
type ContentSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

// IntegrationSummary is the lightweight response for list operations.
type IntegrationSummary struct {
	Manifest `json:"manifest"`
	Overview string `json:"overview"`
}

// Integration is the full detail response, including sections parsed from markdown files.
type Integration struct {
	IntegrationSummary
	Sections []ContentSection `json:"sections"`
}

// CategoryCount represents the count of integrations per category.
type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}
