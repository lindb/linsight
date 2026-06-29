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

package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/lindb/common/pkg/logger"
)

var compositeLog = logger.GetLogger("agent", "composite-tools")

// ToolsProvider is the minimal interface that SkillRegistry uses to load the
// org's available tool set. Both MCPToolsFactory and BuiltinToolsFactory
// implement this interface without any code changes.
type ToolsProvider interface {
	GetToolsForOrg(ctx context.Context, orgID int64) ([]tool.BaseTool, error)
}

// CompositeToolsFactory aggregates multiple ToolsProvider implementations.
// Providers are queried in order; failures are fail-open (logged and skipped)
// so one unavailable provider never blocks the others.
type CompositeToolsFactory struct {
	providers []ToolsProvider
}

// NewCompositeToolsFactory creates a CompositeToolsFactory from the given providers.
func NewCompositeToolsFactory(providers ...ToolsProvider) *CompositeToolsFactory {
	return &CompositeToolsFactory{providers: providers}
}

// GetToolsForOrg merges the tool lists from all registered providers.
// If a provider returns an error it is logged and skipped.
func (c *CompositeToolsFactory) GetToolsForOrg(ctx context.Context, orgID int64) ([]tool.BaseTool, error) {
	var result []tool.BaseTool
	for _, p := range c.providers {
		tools, err := p.GetToolsForOrg(ctx, orgID)
		if err != nil {
			compositeLog.Warn("tools provider failed, skipping", logger.Error(err))
			continue
		}
		result = append(result, tools...)
	}
	return result, nil
}
