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

package model

import (
	"encoding/json"
	"time"

	"github.com/lindb/common/pkg/encoding"
	"gorm.io/datatypes"
)

// AlertEventState represents the lifecycle state of an alert event.
type AlertEventState int

const (
	AlertEventPending  AlertEventState = iota + 1 // condition is true but not yet firing (waiting for 'for' duration)
	AlertEventFiring                               // condition has been true long enough; alert is active
	AlertEventResolved                             // condition is no longer true; alert has ended
	AlertEventNoData                               // query returned no data
	AlertEventError                                // query or evaluation encountered an error
)

// MatchType represents the operator used in a label matcher.
type MatchType string

const (
	MatchEqual     MatchType = "="
	MatchNotEqual  MatchType = "!="
	MatchRegexp    MatchType = "=~"
	MatchNotRegexp MatchType = "!~"
)

// LabelMatcher defines a single label matching rule for routing/silencing/inhibiting.
type LabelMatcher struct {
	Name  string    `json:"name"`  // label key
	Value string    `json:"value"` // match value or regex pattern
	Type  MatchType `json:"type"`  // matching operator
}

// AlertEvent represents the state of a single alert instance identified by its fingerprint.
type AlertEvent struct {
	BaseModel

	OrgID        int64           `json:"-" gorm:"column:org_id;index:idx_alert_event_org"`
	UID          string          `json:"uid" gorm:"column:uid;index:u_idx_alert_event_uid,unique"`
	RuleUID      string          `json:"ruleUID" gorm:"column:rule_uid;index:idx_alert_event_rule"`
	RuleTitle    string          `json:"ruleTitle" gorm:"column:rule_title"`
	Fingerprint  string          `json:"fingerprint" gorm:"column:fingerprint;index:idx_alert_event_fp"`
	State        AlertEventState `json:"state" gorm:"column:state;index:idx_alert_event_state"`
	Severity     Severity        `json:"severity" gorm:"column:severity"`
	Labels       datatypes.JSON  `json:"labels,omitempty" gorm:"column:labels"`
	Annotations  datatypes.JSON  `json:"annotations,omitempty" gorm:"column:annotations"`
	Value        float64         `json:"value" gorm:"column:value"`
	StartsAt     int64           `json:"startsAt" gorm:"column:starts_at"`
	EndsAt       int64           `json:"endsAt" gorm:"column:ends_at"`
	PendingSince int64           `json:"pendingSince" gorm:"column:pending_since"`
	NotifiedAt   int64           `json:"notifiedAt" gorm:"column:notified_at"`
	GeneratorURL string          `json:"generatorURL" gorm:"column:generator_url"`
}

// AlertRoute represents a node in the alert routing tree.
type AlertRoute struct {
	BaseModel

	OrgID          int64          `json:"-" gorm:"column:org_id"`
	UID            string         `json:"uid" gorm:"column:uid;index:u_idx_alert_route_uid,unique"`
	Name           string         `json:"name" gorm:"column:name"`
	ParentUID      string         `json:"parentUID" gorm:"column:parent_uid"`
	Matchers       datatypes.JSON `json:"matchers,omitempty" gorm:"column:matchers"`
	ReceiverUID    string         `json:"receiverUID" gorm:"column:receiver_uid"`
	Continue       bool           `json:"continue" gorm:"column:continue_match"`
	GroupBy        datatypes.JSON `json:"groupBy,omitempty" gorm:"column:group_by"`
	GroupWait      time.Duration  `json:"groupWait" gorm:"column:group_wait"`
	GroupInterval  time.Duration  `json:"groupInterval" gorm:"column:group_interval"`
	RepeatInterval time.Duration  `json:"repeatInterval" gorm:"column:repeat_interval"`
	SortOrder      int            `json:"sortOrder" gorm:"column:sort_order"`
}

func (r *AlertRoute) ToMatchers() ([]LabelMatcher, error) {
	var rs []LabelMatcher
	data, err := r.Matchers.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *AlertRoute) ToGroupBy() ([]string, error) {
	var rs []string
	data, err := r.GroupBy.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// ReceiverChannel represents a single notification channel within a receiver.
type ReceiverChannel struct {
	Type    string          `json:"type"`    // slack / email / webhook / dingtalk / feishu
	Config  json.RawMessage `json:"config"`  // channel-specific configuration
	Enabled bool            `json:"enabled"` // whether this channel is active
}

// AlertReceiver represents a named set of notification channels.
type AlertReceiver struct {
	BaseModel

	OrgID       int64          `json:"-" gorm:"column:org_id"`
	UID         string         `json:"uid" gorm:"column:uid;index:u_idx_alert_receiver_uid,unique"`
	Name        string         `json:"name" gorm:"column:name"`
	Channels    datatypes.JSON `json:"channels,omitempty" gorm:"column:channels"`
	TemplateUID string         `json:"templateUID" gorm:"column:template_uid"`
}

func (r *AlertReceiver) ToChannels() ([]ReceiverChannel, error) {
	var rs []ReceiverChannel
	data, err := r.Channels.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// AlertSilence suppresses notifications for alerts matching the given label matchers during the specified time window.
type AlertSilence struct {
	BaseModel

	OrgID         int64          `json:"-" gorm:"column:org_id;index:idx_alert_silence_org"`
	UID           string         `json:"uid" gorm:"column:uid;index:u_idx_alert_silence_uid,unique"`
	Matchers      datatypes.JSON `json:"matchers,omitempty" gorm:"column:matchers"`
	StartsAt      int64          `json:"startsAt" gorm:"column:starts_at"`
	EndsAt        int64          `json:"endsAt" gorm:"column:ends_at"`
	CreatedByName string         `json:"createdByName" gorm:"column:created_by_name"`
	Comment       string         `json:"comment" gorm:"column:comment"`
}

func (s *AlertSilence) ToMatchers() ([]LabelMatcher, error) {
	var rs []LabelMatcher
	data, err := s.Matchers.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// AlertInhibitRule suppresses target alerts when source alerts are firing and share the same equal-label values.
type AlertInhibitRule struct {
	BaseModel

	OrgID          int64          `json:"-" gorm:"column:org_id"`
	UID            string         `json:"uid" gorm:"column:uid;index:u_idx_alert_inhibit_uid,unique"`
	Name           string         `json:"name" gorm:"column:name"`
	SourceMatchers datatypes.JSON `json:"sourceMatchers,omitempty" gorm:"column:source_matchers"`
	TargetMatchers datatypes.JSON `json:"targetMatchers,omitempty" gorm:"column:target_matchers"`
	EqualLabels    datatypes.JSON `json:"equalLabels,omitempty" gorm:"column:equal_labels"`
	Enabled        bool           `json:"enabled" gorm:"column:enabled"`
}

func (r *AlertInhibitRule) ToEqualLabels() ([]string, error) {
	var rs []string
	data, err := r.EqualLabels.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// ToSourceMatchers decodes the SourceMatchers JSON field.
func (r *AlertInhibitRule) ToSourceMatchers() ([]LabelMatcher, error) {
	var rs []LabelMatcher
	data, err := r.SourceMatchers.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// ToTargetMatchers decodes the TargetMatchers JSON field.
func (r *AlertInhibitRule) ToTargetMatchers() ([]LabelMatcher, error) {
	var rs []LabelMatcher
	data, err := r.TargetMatchers.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

// NotificationTemplate stores a reusable Go template string for rendering alert notifications.
// Templates may contain named {{define "name"}} blocks that callers can invoke with {{template "name" .}}.
type NotificationTemplate struct {
	BaseModel

	OrgID    int64  `json:"-" gorm:"column:org_id"`
	UID      string `json:"uid" gorm:"column:uid;index:u_idx_notif_tmpl_uid,unique"`
	Name     string `json:"name" gorm:"column:name"`
	Template string `json:"template" gorm:"column:template;type:text"`
}

// NotificationTemplateExample stores a reusable sample alert payload used to preview or test notification templates.
// The Payload field holds a JSON-serialised NotificationData value.
type NotificationTemplateExample struct {
	BaseModel

	OrgID   int64  `json:"-" gorm:"column:org_id"`
	UID     string `json:"uid" gorm:"column:uid;index:u_idx_notif_tmpl_example_uid,unique"`
	Name    string `json:"name" gorm:"column:name"`
	Payload string `json:"payload" gorm:"column:payload;type:text"` // JSON of template.NotificationData
}

// SearchAlertEventRequest holds parameters for querying alert events.
type SearchAlertEventRequest struct {
	PagingParam
	RuleUID string `form:"ruleUID" json:"ruleUID"`
	State   int    `form:"state" json:"state"`
}

// SearchAlertRouteRequest holds parameters for querying alert routes.
type SearchAlertRouteRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

// SearchAlertReceiverRequest holds parameters for querying alert receivers.
type SearchAlertReceiverRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

// SearchAlertSilenceRequest holds parameters for querying alert silences.
type SearchAlertSilenceRequest struct {
	PagingParam
}

// SearchAlertInhibitRuleRequest holds parameters for querying alert inhibit rules.
type SearchAlertInhibitRuleRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

// SearchNotificationTemplateRequest holds parameters for querying notification templates.
type SearchNotificationTemplateRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

// SearchNotificationTemplateExampleRequest holds parameters for querying notification template examples.
type SearchNotificationTemplateExampleRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

type AlertRuleState int

const (
	RuleEnable AlertRuleState = iota + 1
	RuleRunning
	RuleFiring
	RulePause
	RuleDelete
)

type Severity int

const (
	AlertSeverity = iota + 1
	WarningSeverity
)

type NotificationSetting struct {
	BaseModel

	OrgID  int64          `json:"-" gorm:"column:org_id;index:u_idx_alert_n_setting,unique"`
	Type   string         `json:"type" gorm:"column:type;index:u_idx_alert_n_setting,unique"`
	Config datatypes.JSON `json:"config,omitempty" gorm:"column:config"`
}

type Notification struct {
	BaseModel

	OrgID     int64          `json:"-" gorm:"column:org_id;index:u_idx_alert_notify,unique"`
	UID       string         `json:"uid" gorm:"column:uid;index:u_idx_alert_notify_uid,unique"`
	Name      string         `json:"name" gorm:"column:name;index:u_idx_alert_notify,unique"`
	Receviers datatypes.JSON `json:"receivers,omitempty" gorm:"column:receivers"`
}

func (n *Notification) ToReceviers() ([]Recevier, error) {
	var rs []Recevier
	data, err := n.Receviers.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

type Recevier struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

// AlertRuleGroup represents a named group of alert rules sharing the same evaluation interval.
// Within the same org, the (org_id, name) pair must be unique.
type AlertRuleGroup struct {
	BaseModel

	OrgID    int64         `json:"-"        gorm:"column:org_id;index:u_idx_alert_rule_group_org_name,unique"`
	UID      string        `json:"uid"      gorm:"column:uid;index:u_idx_alert_rule_group_uid,unique"`
	Name     string        `json:"name"     gorm:"column:name;index:u_idx_alert_rule_group_org_name,unique"`
	Interval time.Duration `json:"interval" gorm:"column:interval"`
}

// CreateAlertRuleGroupParam carries group metadata when creating a rule.
// If a group with the same name already exists in the org, it is reused; otherwise a new group is created.
type CreateAlertRuleGroupParam struct {
	Name     string        `json:"name"`     // required; used as find-or-create key
	Interval time.Duration `json:"interval"` // required only when the group does not yet exist
}

// CreateAlertRuleRequest combines an AlertRule with the group it should belong to.
type CreateAlertRuleRequest struct {
	AlertRule
	Group CreateAlertRuleGroupParam `json:"group"` // always required
}

// GroupWithRules is the unit returned by SearchAlertRules: one group together with its matching rules.
type GroupWithRules struct {
	AlertRuleGroup
	Rules []AlertRule `json:"rules"`
}

// SearchAlertRuleGroupRequest holds parameters for querying alert rule groups.
type SearchAlertRuleGroupRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

type AlertRule struct {
	BaseModel

	OrgID         int64          `json:"-" gorm:"column:org_id"`
	UID           string         `json:"uid" gorm:"column:uid;index:u_idx_alert_rule_uid,unique"`
	GroupUID      string         `json:"groupUID"      gorm:"column:group_uid;index:idx_alert_rule_group_uid"`
	DatasourceUID string         `json:"datasourceUID" gorm:"column:datasource_uid;index:idx_alert_rule_datasource_uid"`
	Title         string         `json:"title" gorm:"column:Title"`
	Integration   string         `json:"integration" gorm:"column:integration"`
	Version       int            `json:"version" gorm:"column:version"`
	State         AlertRuleState `json:"state" gorm:"column:state"`
	Interval      time.Duration  `json:"interval" gorm:"column:interval"`
	For           time.Duration  `json:"for" gorm:"column:for_duration"`  // pending duration before firing
	Data          datatypes.JSON `json:"data,omitempty" gorm:"column:data"`
	Conditions    datatypes.JSON `json:"conditions,omitempty" gorm:"column:conditions"`
	Notifications datatypes.JSON `json:"notifications,omitempty" gorm:"column:notifications"` // kept for backward compat
	Labels        datatypes.JSON `json:"labels,omitempty" gorm:"column:labels"`               // static labels attached to all alert events
	Annotations   datatypes.JSON `json:"annotations,omitempty" gorm:"column:annotations"`     // Go-template annotations
}

func (r *AlertRule) ToNotificatins() ([]string, error) {
	var rs []string
	data, err := r.Notifications.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *AlertRule) ToConditions() ([]Condition, error) {
	var rs []Condition
	data, err := r.Conditions.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := encoding.JSONUnmarshal(data, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *AlertRule) ToDataQuery() (*QueryRequest, error) {
	data, err := r.Data.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Unwrap one level of string encoding when the column value was stored double-encoded
	// (e.g. the frontend serialised the queries array to a JSON string before sending it).
	if len(data) > 0 && data[0] == '"' {
		var inner string
		if err := encoding.JSONUnmarshal(data, &inner); err != nil {
			return nil, err
		}
		data = []byte(inner)
	}

	// Legacy / frontend format: just the queries array "[{...}]" without a Range wrapper.
	// Wrap it in a QueryRequest so the rest of the pipeline stays unchanged.
	if len(data) > 0 && data[0] == '[' {
		var queries []*Query
		if err := encoding.JSONUnmarshal(data, &queries); err != nil {
			return nil, err
		}
		return &QueryRequest{Queries: queries}, nil
	}

	var req *QueryRequest
	if err := encoding.JSONUnmarshal(data, &req); err != nil {
		return nil, err
	}
	return req, nil
}

type Condition struct {
	Severity Severity        `json:"severity"`
	Expr     json.RawMessage `json:"expr"`
}

type SearchAlertRuleRequest struct {
	PagingParam
	Title         string `form:"title"         json:"title"`
	GroupName     string `form:"groupName"     json:"groupName"`
	DatasourceUID string `form:"datasourceUID" json:"datasourceUID"`
}

type Alert struct {
	BaseModel
	OrgID     int64       `json:"-" gorm:"column:org_id"`
	UID       string      `json:"uid" gorm:"column:uid"`
	RuleUID   string      `json:"rule_uid" gorm:"column:rule_uid;index:u_idx_alert_uid,unique"`
	DataPoint *DataPoint  `gorm:"-"`
}

type ReceiverPlugin struct {
	Type    string          `json:"type"`
	Setting json.RawMessage `json:"setting"`
	Config  json.RawMessage `json:"config"`
	Options json.RawMessage `json:"options,omitempty"`
}

type SearchNotificationRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}

// DataPoint represents a single evaluated data point extracted from a datasource result row.
// All numeric columns from the same row are merged into Values so that multi-field
// alert conditions (e.g. "cpu_usage > 80 && memory_usage > 90") can be evaluated atomically.
type DataPoint struct {
	RefID     string
	Labels    map[string]string  // GROUP-BY dimensions → fingerprint + AlertEvent.Labels
	Values    map[string]float64 // all numeric columns from the same row
	Fields    map[string]any     // non-dimension, non-numeric columns (bool, JSON, etc.); reserved for log/event alerting
	Timestamp int64              // for time_series: point timestamp; for scalar: Range.End
	Value     float64            // representative value: Values["value"] ?? first numeric col
	Severity  Severity           // set during Eval when a condition matches; not persisted
}

func (dp *DataPoint) ToMap() map[string]any {
	env := map[string]any{
		"refId":     dp.RefID,
		"value":     dp.Value,
		"timestamp": dp.Timestamp,
	}
	// Lowest priority: dimension labels
	for k, v := range dp.Labels {
		env[k] = v
	}
	// Medium priority: non-dimension content fields
	for k, v := range dp.Fields {
		env[k] = v
	}
	// Highest priority: numeric values (override same-named label/field)
	for k, v := range dp.Values {
		env[k] = v
	}
	return env
}

// AlertRuleDryRunResult captures the full output of a single dry-run evaluation cycle.
// When a pipeline stage fails, Stage and Error are set; DataPoints and Events are empty.
// When all stages succeed, DataPoints contains the raw datasource data and Events contains
// the alert events that would have been produced (but were NOT persisted).
type AlertRuleDryRunResult struct {
	Stage      string        `json:"stage,omitempty"`      // prepare / fetch / eval — set only on failure
	Error      string        `json:"error,omitempty"`      // error message — set only on failure
	DataPoints []*DataPoint  `json:"dataPoints,omitempty"` // raw data points returned by the datasource
	Events     []*AlertEvent `json:"events,omitempty"`     // alert events from the in-memory state machine
}
