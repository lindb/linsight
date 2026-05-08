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

// TemplateType represents the kind of notification template.
type TemplateType string

const (
	TemplateTitleType TemplateType = "title"
	TemplateBodyType  TemplateType = "body"
	TemplateHTMLType  TemplateType = "html"
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
type NotificationTemplate struct {
	BaseModel

	OrgID    int64        `json:"-" gorm:"column:org_id"`
	UID      string       `json:"uid" gorm:"column:uid;index:u_idx_notif_tmpl_uid,unique"`
	Name     string       `json:"name" gorm:"column:name"`
	Type     TemplateType `json:"type" gorm:"column:type"`
	Template string       `json:"template" gorm:"column:template;type:text"`
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

type AlertRule struct {
	BaseModel

	OrgID         int64          `json:"-" gorm:"column:org_id"`
	UID           string         `json:"uid" gorm:"column:uid;index:u_idx_alert_rule_uid,unique"`
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

func (r *AlertRule) ToDataQuery() (req *QueryRequest, err error) {
	data, err := r.Data.MarshalJSON()
	if err != nil {
		return nil, err
	}
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
	Title string `form:"title" json:"title"`
}

type Alert struct {
	BaseModel
	OrgID   int64   `json:"-" gorm:"column:org_id"`
	UID     string  `json:"uid" gorm:"column:uid"`
	RuleUID string  `json:"rule_uid" gorm:"column:rule_uid;index:u_idx_alert_uid,unique"`
	Series  *Series `gorm:"-"`
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

type Series struct {
	RefID      string
	MetricName string
	Field      string
	Tags       map[string]string
	Timestamp  int64
	Value      float64
}

func (s *Series) ToMap() map[string]any {
	return map[string]any{
		"refId":     s.RefID,
		"metric":    s.MetricName,
		"tags":      s.Tags,
		"labels":    s.Tags,
		"timestamp": s.Timestamp,
		s.Field:     s.Value,
	}
}
