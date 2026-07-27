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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"
)

const toolNameIntegrationDiscover = "integration_discover"

const toolDescriptionBaseIntegrationDiscover = `Discover relevant integrations for AIOps root cause analysis (RCA)

<discover_instructions>
When diagnosing a system issue or anomaly, use this tool to discover which integrations (monitoring systems, databases, container orchestrators) are relevant to your diagnosis.

The available integrations are listed below. Read through them to understand what diagnostic data each provides.

How to invoke:
- Examine the <available_integrations> list to identify which ones are relevant to the issue
- Call this tool with a JSON object: {"integrations": [{"id": "...", "sections": ["..."]}, ...]}
- For each selected integration, include the sections you need:
  - "monitoring": standard metrics names, log search patterns, diagnostic thresholds
  - "setup": installation and configuration instructions
  - "overview": general information about the integration

Important:
- Only select integrations that are relevant to the symptoms or technical stack
- Specify which sections you need to minimize context size
- After discovering integrations, use integration_load to get the detailed content for each
</discover_instructions>
`

// integrationSummaryTemplate renders the <available_integrations> block.
var integrationSummaryTemplate = template.Must(template.New("integrations").Parse(`
<available_integrations>
{{- range . }}
<integration>
<id>{{ .ID }}</id>
<name>{{ .Name }}</name>
<category>{{ .Category }}</category>
<description>{{ .Description }}</description>
{{- if .Symptoms }}
<symptoms>
{{- range .Symptoms }}
  - {{ . }}
{{- end }}
</symptoms>
{{- end }}
</integration>
{{- end }}
</available_integrations>
`))

type integrationDiscoverInput struct {
	Integrations []struct {
		ID       string   `json:"id"`
		Sections []string `json:"sections"`
	} `json:"integrations"`
}

type integrationSpec struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Sections []string `json:"sections"`
}

// integrationDiscoverTool implements tool.BaseTool with dynamic Info() rendering.
type integrationDiscoverTool struct {
	catalogSrv catalogReader
	summaries  interface{} // cached list of IntegrationSummary (from last ListCategories or SearchIntegrations)
}

func NewIntegrationDiscoverTool(catalogSrv catalogReader) (tool.BaseTool, error) {
	return &integrationDiscoverTool{catalogSrv: catalogSrv}, nil
}

// Info dynamically renders all available integrations.
func (t *integrationDiscoverTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	summaries, err := t.catalogSrv.SearchIntegrations(ctx, "", "")
	if err != nil {
		toolsLog.Warn("integration_discover: failed to list integrations",
			logger.Error(err))
		summaries = nil
	}

	var buf bytes.Buffer
	if summaries != nil {
		if err := integrationSummaryTemplate.Execute(&buf, summaries); err != nil {
			return nil, fmt.Errorf("integration_discover: render integrations: %w", err)
		}
	}

	return &schema.ToolInfo{
		Name: toolNameIntegrationDiscover,
		Desc: toolDescriptionBaseIntegrationDiscover + buf.String(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"integrations": {
				Type: schema.Array,
				Desc: "List of integrations to load, each with id and sections array. " +
					"Example: [{\"id\":\"kubernetes\",\"sections\":[\"monitoring\",\"setup\"]}]",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun validates and returns the selected integrations.
func (t *integrationDiscoverTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := &integrationDiscoverInput{}
	if err := json.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", fmt.Errorf("integration_discover: unmarshal arguments: %w", err)
	}

	if len(args.Integrations) == 0 {
		return "No integrations selected for discovery.", nil
	}

	result := make([]integrationSpec, 0, len(args.Integrations))
	for _, selected := range args.Integrations {
		// Validate by attempting to load each integration
		integ, err := t.catalogSrv.GetIntegration(ctx, selected.ID)
		if err != nil {
			toolsLog.Warn("integration_discover: integration not found",
				logger.String("id", selected.ID), logger.Error(err))
			continue // skip invalid integrations
		}

		spec := integrationSpec{
			ID:       selected.ID,
			Name:     integ.Manifest.Name,
			Category: integ.Manifest.Category,
			Sections: selected.Sections,
		}
		result = append(result, spec)
	}

	if len(result) == 0 {
		return "Selected integrations not found or not available.", nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return fmt.Sprintf("Integrations discovered:\n%s", string(data)), nil
}

func init() {
	register(func(deps BuiltinDeps) (tool.BaseTool, error) {
		return NewIntegrationDiscoverTool(deps.CatalogSrv)
	})
}
