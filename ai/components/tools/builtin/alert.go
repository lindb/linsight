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

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/datatypes"

	toolscomp "github.com/lindb/linsight/ai/components/tools"
	"github.com/lindb/linsight/model"
)

func init() {
	register(func(deps BuiltinDeps) (tool.BaseTool, error) {
		t, err := toolutils.InferTool(
			"linsight_create_alert_rule",
			"Create an alert rule that fires when a metric/log query crosses a threshold. "+
				"The rule is grouped under a named alert-rule group. "+
				"Requires human approval before it is persisted.",
			func(ctx context.Context, req createAlertRuleRequest) (*toolscomp.ToolCallResult, error) {
				return executeCreateAlertRule(ctx, deps.AlertRuleSrv, &req)
			},
		)
		if err != nil {
			return nil, err
		}
		return withExtraInfo(t, map[string]any{
			"icon":        "alert",
			"render_type": "AlertRuleCard",
			// _gov_need_approval tells the interruptCheck node to pause and await human sign-off.
			// The _gov_ prefix is stripped by getToolPublicExtra before reaching the frontend.
			"_gov_need_approval": true,
			// _gov_card_type hints to the frontend which card component to render in the
			// approval UI (a structured AlertRuleCard instead of raw JSON).
			"_gov_card_type": "alert_rule_create",
		}), nil
	})
}

// createAlertRuleRequest is the input schema inferred by toolutils.InferTool.
// Field names and json tags become the tool's JSON schema for the LLM. The schema
// intentionally exposes only the fields an LLM can reasonably fill; server-managed
// fields (uid, org_id, version, state) are set during persistence.
type createAlertRuleRequest struct {
	// Human-readable name for the alert rule.
	// The jsonschema description guides the LLM to produce a good title at generation
	// time (concise; derived from the conversation context; in the user's language),
	// which is preferable to regenerating it after the fact. Avoid commas in the
	// description string: the jsonschema tag is comma-separated and a comma would be
	// parsed as the start of a new tag.
	Title string `json:"title" jsonschema:"description=Concise human-readable name for the alert rule (max ~60 characters). Derive it from the conversation context (the metric/condition being alerted on) and write it in the SAME language as the user's request. No surrounding quotes or trailing punctuation."`
	// Name of the alert-rule group to place this rule in; created if it does not exist.
	GroupName string `json:"group_name" jsonschema:"description=Name of the alert-rule group; created if missing"`
	// UID of the data source to query (discover via linsight_list_datasources).
	DatasourceUID string `json:"datasource_uid" jsonschema:"description=UID of the data source to query"`
	// Integration type this rule targets, e.g. kubernetes.
	Integration string `json:"integration,omitempty" jsonschema:"description=Integration type, e.g. kubernetes"`
	// The metric or log query (e.g. SQL) that produces the alerting signal.
	Query string `json:"query" jsonschema:"description=The metric or log query (e.g. SQL) producing the signal"`
	// Human-readable summary annotation shown when the alert fires.
	Summary string `json:"summary,omitempty" jsonschema:"description=Summary annotation shown when the alert fires"`
	// Static labels attached to every alert event, e.g. {"env":"test"}.
	Labels map[string]string `json:"labels,omitempty" jsonschema:"description=Static labels attached to every alert event"`
}

// alertRuleData mirrors the {queries,range} JSON stored in AlertRule.Data. The LLM
// supplies only a query string; we wrap it in the datasource-query envelope the
// alerting engine expects.
type alertRuleData struct {
	Queries []alertRuleQuery `json:"queries"`
	Range   alertRuleRange   `json:"range"`
}

type alertRuleQuery struct {
	RefID      string               `json:"refId"`
	Datasource alertRuleDatasource  `json:"datasource"`
	ShowField  bool                 `json:"showField"`
	Params     alertRuleQueryParams `json:"params"`
}

type alertRuleDatasource struct {
	UID  string `json:"uid"`
	Type string `json:"type"`
}

type alertRuleQueryParams struct {
	SQL string `json:"sql"`
}

type alertRuleRange struct {
	Start string `json:"start"`
}

// executeCreateAlertRule builds a full CreateAlertRuleRequest from the LLM-supplied
// fields and persists it through the alert-rule service. It runs only after the
// human approval gate (see the interruptCheck node), so reaching this point means
// the invocation was explicitly authorized.
func executeCreateAlertRule(
	ctx context.Context,
	srv alertRuleCreator,
	req *createAlertRuleRequest,
) (*toolscomp.ToolCallResult, error) {
	if srv == nil {
		return nil, fmt.Errorf("alert rule service is not configured")
	}
	if req.Title == "" || req.GroupName == "" || req.DatasourceUID == "" {
		return nil, fmt.Errorf("title, group_name and datasource_uid are required")
	}

	// Wrap the raw query into the datasource-query envelope stored in AlertRule.Data.
	dataEnvelope := alertRuleData{
		Queries: []alertRuleQuery{{
			RefID:      "A",
			Datasource: alertRuleDatasource{UID: req.DatasourceUID, Type: "lindb"},
			ShowField:  true,
			Params:     alertRuleQueryParams{SQL: req.Query},
		}},
		Range: alertRuleRange{Start: "now-15m"},
	}
	dataJSON, err := json.Marshal(dataEnvelope)
	if err != nil {
		return nil, fmt.Errorf("marshal alert rule data: %w", err)
	}

	createReq := &model.CreateAlertRuleRequest{
		AlertRule: model.AlertRule{
			Title:         req.Title,
			DatasourceUID: req.DatasourceUID,
			Integration:   req.Integration,
			State:         model.RuleEnable,
			Data:          datatypes.JSON(dataJSON),
		},
		Group: model.CreateAlertRuleGroupParam{Name: req.GroupName},
	}
	if len(req.Labels) > 0 {
		labelsJSON, lErr := json.Marshal(req.Labels)
		if lErr != nil {
			return nil, fmt.Errorf("marshal alert rule labels: %w", lErr)
		}
		createReq.Labels = datatypes.JSON(labelsJSON)
	}
	if req.Summary != "" {
		annJSON, aErr := json.Marshal(map[string]string{"summary": req.Summary})
		if aErr != nil {
			return nil, fmt.Errorf("marshal alert rule annotations: %w", aErr)
		}
		createReq.Annotations = datatypes.JSON(annJSON)
	}

	// Persist. The service reads org/user from ctx (util.GetUser), so the auth
	// context threaded from the chat request must be present here.
	uid, err := srv.CreateAlertRule(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("create alert rule: %w", err)
	}

	summary := map[string]any{
		"uid":            uid,
		"title":          req.Title,
		"group_name":     req.GroupName,
		"datasource_uid": req.DatasourceUID,
		"status":         "created",
	}
	data, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshal alert rule result: %w", err)
	}
	return &toolscomp.ToolCallResult{
		Data:  data,
		Extra: map[string]any{"uid": uid, "title": req.Title},
	}, nil
}
