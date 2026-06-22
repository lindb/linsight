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

package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

// DryRunner executes the full alert rule evaluation pipeline once without persistent side effects.
// By default (send=false) no notifications are dispatched and no DB writes occur.
// With send=true the real notification channels are called.
type DryRunner interface {
	Run(ctx context.Context, rule *model.AlertRule, send bool) (*model.AlertRuleDryRunResult, error)
}

type dryRunner struct {
	deps *Deps
}

// NewDryRunner creates a DryRunner backed by the given scheduling dependencies.
func NewDryRunner(deps *Deps) DryRunner {
	return &dryRunner{deps: deps}
}

// Run executes the full pipeline for the given rule and returns a structured diagnostic result.
// Pipeline stages: Prepare → FetchData → EvalAndTrack → SilenceCheck → InhibitCheck → RouteMatch → ChannelRender.
// Errors at each early stage are captured inside the result (Stage + Error fields).
func (d *dryRunner) Run(ctx context.Context, rule *model.AlertRule, send bool) (*model.AlertRuleDryRunResult, error) {
	alertRule := NewAlertRule(rule, d.deps)

	// Stage 1: compile conditions and decode query parameters.
	if err := alertRule.Perpare(); err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "prepare",
			Error: err.Error(),
		}, nil
	}

	// Stage 2: fetch raw series from the configured datasource.
	seriesList, err := alertRule.FetchData()
	if err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "fetch",
			Error: err.Error(),
		}, nil
	}

	// Stage 3: evaluate conditions and run the in-memory state machine.
	// EvalAndTrack uses a fresh RuleStateManager (no DB), so no events are persisted.
	events, err := alertRule.EvalAndTrack(ctx, seriesList, time.Now())
	if err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "eval",
			Error: err.Error(),
		}, nil
	}

	result := &model.AlertRuleDryRunResult{
		DataPoints: seriesList,
		Events:     events,
	}

	// Build per-query info (datasource metadata + grouped data points).
	result.Queries = d.buildQueryInfo(rule, alertRule.queryParams, seriesList)

	// Simulate the AlertManager pipeline when ManagerDeps is available.
	if d.deps.ManagerDeps != nil && len(events) > 0 {
		result.EventResults = d.simulatePipeline(ctx, rule.OrgID, events, send)
	}

	return result, nil
}

// buildQueryInfo constructs DryRunQueryInfo for each query in the request,
// grouping the returned data points by refID.
func (d *dryRunner) buildQueryInfo(rule *model.AlertRule, params *model.QueryRequest, dataPoints []*model.DataPoint) []model.DryRunQueryInfo {
	if params == nil || len(params.Queries) == 0 {
		return nil
	}

	// Group data points by refID for O(1) lookup below.
	byRefID := make(map[string][]*model.DataPoint, len(dataPoints))
	for _, dp := range dataPoints {
		byRefID[dp.RefID] = append(byRefID[dp.RefID], dp)
	}

	var infos []model.DryRunQueryInfo
	for _, q := range params.Queries {
		info := model.DryRunQueryInfo{
			RefID:         q.RefID,
			DatasourceUID: q.Datasource.UID,
			Query:         q,
			TimeRange:     params.Range,
			DataPoints:    byRefID[q.RefID],
		}
		// Enrich with datasource name and type when the service is available.
		if d.deps.DatasourceSrv != nil {
			if ds, err := d.deps.DatasourceSrv.GetDatasourceByOrgAndUID(rule.OrgID, q.Datasource.UID); err == nil {
				info.DatasourceName = ds.Name
				info.DatasourceType = string(ds.Type)
			}
		}
		infos = append(infos, info)
	}
	return infos
}

// simulatePipeline applies SilenceStage, InhibitStage and RouteStage logic to each event
// and captures full diagnostic details without modifying any shared state.
func (d *dryRunner) simulatePipeline(ctx context.Context, orgID int64, events []*model.AlertEvent, send bool) []model.DryRunEventResult {
	now := time.Now().UnixMilli()
	mgr := d.deps.ManagerDeps

	// Load org-level config snapshots (read-only).
	silences, _ := mgr.AlertSilenceSrv.GetActiveSilencesByOrg(orgID, now)
	inhibits, _ := mgr.AlertInhibitRuleSrv.GetEnabledInhibitRulesByOrg(orgID)
	routeTree, _ := mgr.AlertRouteSrv.GetAlertRouteTreeByOrg(orgID)

	// For inhibit source matching we use only the current eval batch (no DB state in dryrun).
	firingEvents := filterFiring(events)

	results := make([]model.DryRunEventResult, 0, len(events))
	for _, ev := range events {
		evResult := model.DryRunEventResult{Event: ev}

		labels := dryParseEventLabels(ev)

		// ── Silence check ────────────────────────────────────────────────────
		evResult.Silences = d.checkSilences(silences, labels, now)
		for _, sc := range evResult.Silences {
			if sc.Matched {
				evResult.Silenced = true
				break
			}
		}

		// ── Inhibit check ────────────────────────────────────────────────────
		evResult.Inhibits = d.checkInhibits(inhibits, firingEvents, ev, labels)
		for _, ic := range evResult.Inhibits {
			if ic.Matched {
				evResult.Inhibited = true
				break
			}
		}

		// ── Route matching and channel render ────────────────────────────────
		if !evResult.Silenced && !evResult.Inhibited {
			evResult.Routes = d.matchRoutes(ctx, orgID, routeTree, ev, labels, []*model.AlertEvent{ev}, send)
		}

		results = append(results, evResult)
	}
	return results
}

// checkSilences returns a DryRunSilenceCheck for every active silence, marking Matched=true
// for any silence whose matchers cover the event labels.
func (d *dryRunner) checkSilences(silences []*model.AlertSilence, labels map[string]string, now int64) []model.DryRunSilenceCheck {
	if len(silences) == 0 {
		return nil
	}
	checks := make([]model.DryRunSilenceCheck, 0, len(silences))
	for _, s := range silences {
		matchers, err := s.ToMatchers()
		if err != nil {
			matchers = nil
		}
		check := model.DryRunSilenceCheck{
			UID:      s.UID,
			Comment:  s.Comment,
			StartsAt: s.StartsAt,
			EndsAt:   s.EndsAt,
			Matchers: matchers,
		}
		// A silence is active when the event falls inside its time window and all matchers match.
		if s.StartsAt <= now && s.EndsAt > now {
			check.Matched = dryMatchAll(labels, matchers)
		}
		checks = append(checks, check)
	}
	return checks
}

// checkInhibits returns a DryRunInhibitCheck for every inhibit rule, marking Matched=true
// when the rule suppresses this event.
func (d *dryRunner) checkInhibits(inhibits []*model.AlertInhibitRule, firingEvents []*model.AlertEvent, ev *model.AlertEvent, targetLabels map[string]string) []model.DryRunInhibitCheck {
	if len(inhibits) == 0 {
		return nil
	}
	checks := make([]model.DryRunInhibitCheck, 0, len(inhibits))
	for _, rule := range inhibits {
		srcMatchers, _ := rule.ToSourceMatchers()
		tgtMatchers, _ := rule.ToTargetMatchers()
		equalLabels, _ := rule.ToEqualLabels()

		check := model.DryRunInhibitCheck{
			UID:            rule.UID,
			Name:           rule.Name,
			SourceMatchers: srcMatchers,
			TargetMatchers: tgtMatchers,
			EqualLabels:    equalLabels,
		}

		// Check if this event matches the target side of the inhibit rule.
		if dryMatchAll(targetLabels, tgtMatchers) {
			// Look for any firing source alert that satisfies source matchers + equal-labels.
			for _, src := range firingEvents {
				if src.Fingerprint == ev.Fingerprint {
					continue
				}
				srcLabels := dryParseEventLabels(src)
				if !dryMatchAll(srcLabels, srcMatchers) {
					continue
				}
				// All equal-label keys must share the same value between source and target.
				ok := true
				for _, lbl := range equalLabels {
					if srcLabels[lbl] != targetLabels[lbl] {
						ok = false
						break
					}
				}
				if ok {
					check.Matched = true
					break
				}
			}
		}
		checks = append(checks, check)
	}
	return checks
}

// matchRoutes traverses the route tree and, for each matched route, builds the notification
// data and calls DryRun on each enabled channel.
func (d *dryRunner) matchRoutes(ctx context.Context, orgID int64, routeTree *model.AlertRouteNode,
	ev *model.AlertEvent, labels map[string]string, events []*model.AlertEvent, send bool) []model.DryRunRouteMatch {

	if routeTree == nil {
		return nil
	}

	matched := dryTraverseNode(routeTree, labels, nil)
	if len(matched) == 0 {
		return nil
	}

	mgr := d.deps.ManagerDeps
	var routeMatches []model.DryRunRouteMatch

	for _, route := range matched {
		if route.ContactPointUID == "" {
			continue
		}

		cp, err := mgr.ContactPointSrv.GetContactPointByOrgAndUID(orgID, route.ContactPointUID)
		if err != nil {
			continue
		}

		groupLabels := d.buildGroupLabels(route, labels)

		status := "firing"
		if ev.State == model.AlertEventResolved {
			status = "resolved"
		}

		data := template.BuildNotificationData(events, cp.Name, status, mgr.ExternalURL, groupLabels)
		if mgr.RenderingEngine != nil {
			data.OrgTemplates, _ = mgr.RenderingEngine.GetMergedOrgTemplates(orgID)
			data.OrgID = orgID
		}

		channels := d.dispatchChannels(ctx, orgID, cp, data, send)

		routeMatches = append(routeMatches, model.DryRunRouteMatch{
			RouteUID:         route.UID,
			RouteName:        route.Name,
			ContactPointUID:  route.ContactPointUID,
			ContactPointName: cp.Name,
			GroupLabels:      groupLabels,
			Status:           status,
			Channels:         channels,
		})
	}
	return routeMatches
}

// dispatchChannels calls DryRun on each enabled channel of the contact point.
func (d *dryRunner) dispatchChannels(ctx context.Context, orgID int64, cp *model.ContactPoint,
	data *template.NotificationData, send bool) []model.DryRunChannelOutput {

	channels, err := cp.ToChannels()
	if err != nil {
		return nil
	}

	mgr := d.deps.ManagerDeps

	// Load global plugin settings once (reused across channels).
	settings, _ := mgr.NotificationSrv.GetNotificationSettingsByOrg(orgID)

	var outputs []model.DryRunChannelOutput
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}

		out := model.DryRunChannelOutput{
			ChannelUID: ch.UID,
			Type:       ch.Type,
		}

		pluginDef, ok := types.GetNotifierDef(ch.Type)
		if !ok {
			out.Error = fmt.Sprintf("notifier plugin %q not registered", ch.Type)
			outputs = append(outputs, out)
			continue
		}

		settingJSON := d.findSettingJSON(settings, ch.Type)
		notifier, err := pluginDef.NewFn(settingJSON, ch.Config, mgr.RenderingEngine)
		if err != nil {
			out.Error = fmt.Sprintf("create notifier: %v", err)
			outputs = append(outputs, out)
			continue
		}

		dryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, dryErr := notifier.DryRun(dryCtx, data, send)
		cancel()

		if dryErr != nil {
			out.Error = dryErr.Error()
		} else {
			out.Result = result
			out.Sent = send
		}
		outputs = append(outputs, out)
	}
	return outputs
}

// findSettingJSON scans the org-level notification settings for the given plugin type.
func (d *dryRunner) findSettingJSON(settings []model.NotificationSetting, pluginType string) json.RawMessage {
	for _, s := range settings {
		if s.Type == pluginType {
			if data, err := s.Config.MarshalJSON(); err == nil {
				return json.RawMessage(data)
			}
		}
	}
	return json.RawMessage("{}")
}

// buildGroupLabels selects label values according to the route's GroupBy configuration.
func (d *dryRunner) buildGroupLabels(route *model.AlertRoute, labels map[string]string) map[string]string {
	groupBy, _ := route.ToGroupBy()
	if len(groupBy) == 0 {
		return nil
	}
	m := make(map[string]string, len(groupBy))
	for _, k := range groupBy {
		m[k] = labels[k]
	}
	return m
}

// filterFiring returns only Firing-state events from the batch (used as allFiring for inhibit).
func filterFiring(events []*model.AlertEvent) []*model.AlertEvent {
	var result []*model.AlertEvent
	for _, ev := range events {
		if ev.State == model.AlertEventFiring {
			result = append(result, ev)
		}
	}
	return result
}

// ─── Route traversal helpers (replicated from alerting/manager — not exported) ───────────────────

// dryTraverseNode is a read-only copy of manager.traverseNode for use within the schedule package.
func dryTraverseNode(node *model.AlertRouteNode, labels map[string]string, ancestors []*model.AlertRoute) []*model.AlertRoute {
	if node == nil {
		return nil
	}
	// Non-root nodes must match event labels; the root (ParentUID="") always passes.
	if node.ParentUID != "" {
		matchers, err := node.ToMatchers()
		if err != nil || !dryMatchAll(labels, matchers) {
			return nil
		}
	}

	newAncestors := append(ancestors, &node.AlertRoute) //nolint:gocritic

	var result []*model.AlertRoute
	childMatched := false
	for _, child := range node.Routes {
		childResults := dryTraverseNode(child, labels, newAncestors)
		if len(childResults) > 0 {
			result = append(result, childResults...)
			childMatched = true
			if !child.Continue {
				break
			}
		}
	}

	if !childMatched {
		resolved := dryResolveInheritance(&node.AlertRoute, ancestors)
		result = append(result, resolved)
	}
	return result
}

// dryResolveInheritance fills zero-value timing fields from the nearest ancestor.
func dryResolveInheritance(route *model.AlertRoute, ancestors []*model.AlertRoute) *model.AlertRoute {
	resolved := *route
	for i := len(ancestors) - 1; i >= 0; i-- {
		p := ancestors[i]
		if resolved.GroupWait == 0 {
			resolved.GroupWait = p.GroupWait
		}
		if resolved.GroupInterval == 0 {
			resolved.GroupInterval = p.GroupInterval
		}
		if resolved.RepeatInterval == 0 {
			resolved.RepeatInterval = p.RepeatInterval
		}
		if len(resolved.GroupBy) == 0 {
			resolved.GroupBy = p.GroupBy
		}
	}
	return &resolved
}

// dryMatchAll returns true when every matcher in the list matches the label set.
func dryMatchAll(labels map[string]string, matchers []model.LabelMatcher) bool {
	for _, m := range matchers {
		val := labels[m.Name]
		switch m.Type {
		case model.MatchEqual:
			if val != m.Value {
				return false
			}
		case model.MatchNotEqual:
			if val == m.Value {
				return false
			}
		case model.MatchRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if !ok {
				return false
			}
		case model.MatchNotRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if ok {
				return false
			}
		}
	}
	return true
}

// dryParseEventLabels decodes the JSON labels of an AlertEvent into a plain map.
func dryParseEventLabels(ev *model.AlertEvent) map[string]string {
	if len(ev.Labels) == 0 {
		return nil
	}
	result := make(map[string]string)
	_ = json.Unmarshal(ev.Labels, &result)
	return result
}
