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

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type integrationLoadInput struct {
	ID       string   `json:"id" jsonschema:"required,description=Integration ID from integration_discover output"`
	Sections []string `json:"sections" jsonschema:"description=Sections to load (e.g. [\"monitoring\",\"setup\"]). Empty loads all."`
}

type sectionContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

type integrationLoadOutput struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Category string           `json:"category"`
	Sections []sectionContent `json:"sections"`
}

// integrationLoadExecutor loads integration content by ID and filters sections.
func integrationLoadExecutor(ctx context.Context, input integrationLoadInput, catalogSrv catalogReader) (*integrationLoadOutput, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("integration_load: integration_id is required")
	}

	integ, err := catalogSrv.GetIntegration(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("integration_load: integration '%s' not found. It may not be installed or configured", input.ID)
	}

	output := &integrationLoadOutput{
		ID:       input.ID,
		Name:     integ.Manifest.Name,
		Category: integ.Manifest.Category,
	}

	// Filter sections based on requested ones
	for _, section := range integ.Sections {
		// If specific sections requested, filter by title (case-insensitive)
		if len(input.Sections) > 0 {
			titleLower := strings.ToLower(section.Title)
			found := false
			for _, reqSection := range input.Sections {
				if strings.ToLower(reqSection) == titleLower {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		output.Sections = append(output.Sections, sectionContent{
			Title:   section.Title,
			Content: section.Content,
			Order:   section.Order,
		})
	}

	return output, nil
}

// integrationLoadTool is the builtin tool for loading integration content.
type integrationLoadTool struct {
	catalogSrv catalogReader
}

func (t *integrationLoadTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "integration_load",
		Desc: "Load detailed content from an integration discovered via integration_discover. " +
			"You can optionally filter to specific sections (monitoring, setup, overview, etc.).",
		ParamsOneOf: nil, // Let InferTool handle this
	}, nil
}

func (t *integrationLoadTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := &integrationLoadInput{}
	if err := json.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", fmt.Errorf("integration_load: unmarshal arguments: %w", err)
	}

	output, err := integrationLoadExecutor(ctx, *args, t.catalogSrv)
	if err != nil {
		return "", err
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data), nil
}

// NewIntegrationLoadTool creates the integration_load tool.
func NewIntegrationLoadTool(catalogSrv catalogReader) (tool.BaseTool, error) {
	return &integrationLoadTool{catalogSrv: catalogSrv}, nil
}

func init() {
	register(func(deps BuiltinDeps) (tool.BaseTool, error) {
		return NewIntegrationLoadTool(deps.CatalogSrv)
	})
}
