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

package service

import (
	"context"

	agentapi "github.com/lindb/linsight/agent/api"
	agentmodel "github.com/lindb/linsight/agent/model"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ── MCP server settings ───────────────────────────────────────────────────────

// ListMCPServers returns all MCP server entries for the org extracted from ctx,
// ordered by ID ascending so that combinedHash in MCPToolsFactory is deterministic.
func (s *agentSettingsService) ListMCPServers(ctx context.Context) ([]*agentmodel.AgentMCPServer, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var servers []*agentmodel.AgentMCPServer
	if err := s.db.RawDB().Where("org_id = ?", orgID).Order("id asc").Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetMCPServer returns a single MCP server by UID within the org extracted from ctx.
func (s *agentSettingsService) GetMCPServer(ctx context.Context, uid string) (*agentmodel.AgentMCPServer, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var server agentmodel.AgentMCPServer
	if err := s.db.Get(&server, "org_id = ? and uid = ?", orgID, uid); err != nil {
		return nil, err
	}
	return &server, nil
}

// CreateMCPServer inserts a new MCP server configuration built from req.
// Returns the generated UID.
func (s *agentSettingsService) CreateMCPServer(ctx context.Context, req *agentapi.CreateMCPServerRequest) (string, error) {
	user := util.GetUser(ctx)
	server := &agentmodel.AgentMCPServer{
		Name:    req.Name,
		Type:    agentmodel.MCPServerType(req.Type),
		URL:     req.URL,
		Command: req.Command,
		Args:    req.Args,
		Enabled: req.Enabled,
	}
	server.UID = uuid.GenerateShortUUID()
	server.OrgID = user.Org.ID
	server.CreatedBy = user.User.ID
	server.UpdatedBy = user.User.ID
	if err := s.db.Create(server); err != nil {
		return "", err
	}
	return server.UID, nil
}

// DeleteMCPServer removes an MCP server configuration by UID within the org extracted from ctx.
func (s *agentSettingsService) DeleteMCPServer(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Delete(&agentmodel.AgentMCPServer{}, "org_id = ? and uid = ?", orgID, uid)
}
