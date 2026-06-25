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

package handler

import (
	agentconfig "github.com/lindb/linsight/agent/config"
	agentservice "github.com/lindb/linsight/agent/service"
	"github.com/lindb/linsight/agent/service/iface"
)

// AgentHandler handles HTTP requests for the agent module.
type AgentHandler struct {
	cfg         *agentconfig.Config
	agentSrv    agentservice.AgentService
	settingsSrv iface.AgentSettingsService
}

// NewAgentHandler creates an AgentHandler wired to the given dependencies.
func NewAgentHandler(cfg *agentconfig.Config, agentSrv agentservice.AgentService, settingsSrv iface.AgentSettingsService) *AgentHandler {
	return &AgentHandler{
		cfg:         cfg,
		agentSrv:    agentSrv,
		settingsSrv: settingsSrv,
	}
}
