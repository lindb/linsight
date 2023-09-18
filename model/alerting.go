package model

import (
	"encoding/json"
	"time"

	"github.com/lindb/common/pkg/encoding"
	"gorm.io/datatypes"
)

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
	Data          datatypes.JSON `json:"data,omitempty" gorm:"column:data"`
	Conditions    datatypes.JSON `json:"conditions,omitempty" gorm:"column:conditions"`
	Notifications datatypes.JSON `json:"notifications,omitempty" gorm:"column:notifications"`
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
