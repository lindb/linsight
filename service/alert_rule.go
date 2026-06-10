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

//go:generate mockgen -source=./alert_rule.go -destination=./alert_rule_mock.go -package=service

import (
	"context"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// groupRuleRow is a flat scan target for the LEFT JOIN between alert_rule_groups and alert_rules.
// All rule columns are pointer types so GORM can scan SQL NULL (no matching rule) without panicking.
type groupRuleRow struct {
	// group columns — never NULL
	GID        int64         `gorm:"column:g_id"`
	GCreatedAt int64         `gorm:"column:g_created_at"`
	GUpdatedAt int64         `gorm:"column:g_updated_at"`
	GCreatedBy int64         `gorm:"column:g_created_by"`
	GUpdatedBy int64         `gorm:"column:g_updated_by"`
	GUID       string        `gorm:"column:g_uid"`
	GName      string        `gorm:"column:g_name"`
	GInterval  time.Duration `gorm:"column:g_interval"`

	// rule columns — NULL when no rule matches (LEFT JOIN with no WHERE on rule columns)
	RID            *int64         `gorm:"column:r_id"`
	RCreatedAt     *int64         `gorm:"column:r_created_at"`
	RUpdatedAt     *int64         `gorm:"column:r_updated_at"`
	RCreatedBy     *int64         `gorm:"column:r_created_by"`
	RUpdatedBy     *int64         `gorm:"column:r_updated_by"`
	ROrgID         *int64         `gorm:"column:r_org_id"`
	RUID           *string        `gorm:"column:r_uid"`
	RGroupUID      *string        `gorm:"column:r_group_uid"`
	RDatasourceUID *string        `gorm:"column:r_datasource_uid"`
	RTitle         *string        `gorm:"column:r_title"`
	RIntegration   *string        `gorm:"column:r_integration"`
	RVersion       *int           `gorm:"column:r_version"`
	RState         *int           `gorm:"column:r_state"`
	RInterval      *int64         `gorm:"column:r_interval"`
	RFor           *int64         `gorm:"column:r_for_duration"`
	RData          datatypes.JSON `gorm:"column:r_data"`
	RConditions    datatypes.JSON `gorm:"column:r_conditions"`
	RNotifications datatypes.JSON `gorm:"column:r_notifications"`
	RLabels        datatypes.JSON `gorm:"column:r_labels"`
	RAnnotations   datatypes.JSON `gorm:"column:r_annotations"`
}

// groupRuleJoinQuery is the base SELECT…FROM…JOIN clause shared by SearchAlertRules.
// A WHERE clause and ORDER BY are appended at runtime.
const groupRuleJoinQuery = `SELECT
  g.id            AS g_id,
  g.created_at    AS g_created_at,
  g.updated_at    AS g_updated_at,
  g.created_by    AS g_created_by,
  g.updated_by    AS g_updated_by,
  g.uid           AS g_uid,
  g.name          AS g_name,
  g.interval      AS g_interval,
  r.id            AS r_id,
  r.created_at    AS r_created_at,
  r.updated_at    AS r_updated_at,
  r.created_by    AS r_created_by,
  r.updated_by    AS r_updated_by,
  r.org_id        AS r_org_id,
  r.uid           AS r_uid,
  r.group_uid     AS r_group_uid,
  r.datasource_uid AS r_datasource_uid,
  r.Title         AS r_title,
  r.integration   AS r_integration,
  r.version       AS r_version,
  r.state         AS r_state,
  r.interval      AS r_interval,
  r.for_duration  AS r_for_duration,
  r.data          AS r_data,
  r.conditions    AS r_conditions,
  r.notifications AS r_notifications,
  r.labels        AS r_labels,
  r.annotations   AS r_annotations
FROM alert_rule_groups g
LEFT JOIN alert_rules r ON r.group_uid = g.uid AND r.org_id = g.org_id`

// reshapeGroupRuleRows converts a flat JOIN result set into an ordered slice of GroupWithRules.
// Group order follows first-seen order (already sorted by g.id DESC from the query).
func reshapeGroupRuleRows(rows []groupRuleRow) []model.GroupWithRules {
	type entry struct {
		idx  int
		item model.GroupWithRules
	}
	index := make(map[string]*entry, len(rows))
	order := make([]string, 0, len(rows))

	for _, row := range rows {
		e, exists := index[row.GUID]
		if !exists {
			e = &entry{
				idx: len(order),
				item: model.GroupWithRules{
					AlertRuleGroup: model.AlertRuleGroup{
						BaseModel: model.BaseModel{
							ID:        row.GID,
							CreatedAt: row.GCreatedAt,
							UpdatedAt: row.GUpdatedAt,
							CreatedBy: row.GCreatedBy,
							UpdatedBy: row.GUpdatedBy,
						},
						UID:      row.GUID,
						Name:     row.GName,
						Interval: row.GInterval,
					},
					Rules: []model.AlertRule{},
				},
			}
			index[row.GUID] = e
			order = append(order, row.GUID)
		}
		// Only append a rule when there is actually a matched rule row (LEFT JOIN may return NULL).
		if row.RID != nil {
			e.item.Rules = append(e.item.Rules, model.AlertRule{
				BaseModel: model.BaseModel{
					ID:        derefInt64(row.RID),
					CreatedAt: derefInt64(row.RCreatedAt),
					UpdatedAt: derefInt64(row.RUpdatedAt),
					CreatedBy: derefInt64(row.RCreatedBy),
					UpdatedBy: derefInt64(row.RUpdatedBy),
				},
				OrgID:         derefInt64(row.ROrgID),
				UID:           derefStr(row.RUID),
				GroupUID:      derefStr(row.RGroupUID),
				DatasourceUID: derefStr(row.RDatasourceUID),
				Title:         derefStr(row.RTitle),
				Integration:   derefStr(row.RIntegration),
				Version:       derefInt(row.RVersion),
				State:         model.AlertRuleState(derefInt(row.RState)),
				Interval:      time.Duration(derefInt64(row.RInterval)),
				For:           time.Duration(derefInt64(row.RFor)),
				Data:          row.RData,
				Conditions:    row.RConditions,
				Notifications: row.RNotifications,
				Labels:        row.RLabels,
				Annotations:   row.RAnnotations,
			})
		}
	}

	result := make([]model.GroupWithRules, 0, len(order))
	for _, uid := range order {
		result = append(result, index[uid].item)
	}
	return result
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// AlertRuleService manages alert rules.
type AlertRuleService interface {
	// SearchAlertRules searches alert rules grouped by their rule group.
	SearchAlertRules(ctx context.Context, req *model.SearchAlertRuleRequest) (rs []model.GroupWithRules, total int64, err error)
	// CreateAlertRule creates an alert rule, finding or creating the target group in the same transaction.
	CreateAlertRule(ctx context.Context, req *model.CreateAlertRuleRequest) (string, error)
	// UpdateAlertRule updates the alert rule by uid.
	UpdateAlertRule(ctx context.Context, rule *model.AlertRule) error
	// DeleteAlertRuleByUID deletes the alert rule by uid.
	DeleteAlertRuleByUID(ctx context.Context, uid string) error
	// GetAlertRuleByUID returns the alert rule by uid.
	GetAlertRuleByUID(ctx context.Context, uid string) (*model.AlertRule, error)
	// FetchAllAlertRules returns all rules with group interval back-filled for the scheduler.
	FetchAllAlertRules(req *model.SearchAlertRuleRequest) (rs []model.AlertRule, err error)
}

type alertRuleService struct {
	db       dbpkg.DB
	groupSrv AlertRuleGroupService
}

// NewAlertRuleService creates an AlertRuleService instance.
func NewAlertRuleService(db dbpkg.DB, groupSrv AlertRuleGroupService) AlertRuleService {
	return &alertRuleService{
		db:       db,
		groupSrv: groupSrv,
	}
}

// SearchAlertRules returns all matching groups together with their rules.
// A single LEFT JOIN query atomically applies both group-level and rule-level filters,
// avoiding N+1 queries. Pagination is omitted because the data volume per org is small.
// When rule-level filters (Title / DatasourceUID) are present, groups with no matching
// rules are naturally excluded by the WHERE clause (NULL rule columns fail the condition).
func (srv *alertRuleService) SearchAlertRules(ctx context.Context, req *model.SearchAlertRuleRequest) ([]model.GroupWithRules, int64, error) {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID

	conds := []string{"g.org_id=?"}
	params := []any{orgID}
	if req.GroupName != "" {
		conds = append(conds, "g.name like ?")
		params = append(params, req.GroupName+"%")
	}
	if req.Title != "" {
		conds = append(conds, "r.title like ?")
		params = append(params, req.Title+"%")
	}
	if req.DatasourceUID != "" {
		conds = append(conds, "r.datasource_uid=?")
		params = append(params, req.DatasourceUID)
	}

	query := groupRuleJoinQuery + " WHERE " + strings.Join(conds, " and ") + " ORDER BY g.id DESC, r.id ASC"
	var rows []groupRuleRow
	if err := srv.db.ExecRaw(&rows, query, params...); err != nil {
		return nil, 0, err
	}

	result := reshapeGroupRuleRows(rows)
	if len(result) == 0 {
		return nil, 0, nil
	}
	return result, int64(len(result)), nil
}

// CreateAlertRule creates an alert rule inside a transaction.
// The group is found by name or created if it does not exist.
func (srv *alertRuleService) CreateAlertRule(ctx context.Context, req *model.CreateAlertRuleRequest) (string, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID

	var uid string
	err := srv.db.Transaction(func(tx dbpkg.DB) error {
		group, err := srv.groupSrv.FindOrCreateAlertRuleGroup(tx, orgID, userID, &req.Group)
		if err != nil {
			return err
		}

		rule := req.AlertRule
		rule.UID = uuid.GenerateShortUUID()
		rule.OrgID = orgID
		rule.GroupUID = group.UID
		rule.CreatedBy = userID
		rule.UpdatedBy = userID
		if err := tx.Create(&rule); err != nil {
			return err
		}
		uid = rule.UID
		return nil
	})
	return uid, err
}

// UpdateAlertRule updates the alert rule (all mutable fields) by uid.
func (srv *alertRuleService) UpdateAlertRule(ctx context.Context, rule *model.AlertRule) error {
	ruleFromDB, err := srv.GetAlertRuleByUID(ctx, rule.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	ruleFromDB.Title = rule.Title
	ruleFromDB.Data = rule.Data
	ruleFromDB.Conditions = rule.Conditions
	ruleFromDB.Integration = rule.Integration
	ruleFromDB.Notifications = rule.Notifications
	ruleFromDB.GroupUID = rule.GroupUID
	ruleFromDB.DatasourceUID = rule.DatasourceUID
	ruleFromDB.Labels = rule.Labels
	ruleFromDB.Annotations = rule.Annotations
	ruleFromDB.For = rule.For
	ruleFromDB.Version += 1
	ruleFromDB.UpdatedBy = user.User.ID
	return srv.db.Update(ruleFromDB, "uid=? and org_id=?", rule.UID, user.Org.ID)
}

// DeleteAlertRuleByUID deletes the alert rule by uid.
func (srv *alertRuleService) DeleteAlertRuleByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		return tx.Delete(&model.AlertRule{}, "uid=? and org_id=?", uid, orgID)
	})
}

// GetAlertRuleByUID returns the alert rule scoped to the signed-in user's org.
func (srv *alertRuleService) GetAlertRuleByUID(ctx context.Context, uid string) (*model.AlertRule, error) {
	signedUser := util.GetUser(ctx)
	var rs model.AlertRule
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return &rs, nil
}

// FetchAllAlertRules returns all rules and back-fills group interval into each rule
// so the scheduler can use rule.Interval without any changes.
func (srv *alertRuleService) FetchAllAlertRules(_ *model.SearchAlertRuleRequest) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	if err := srv.db.Find(&rules); err != nil {
		return nil, err
	}

	// Build a group-interval lookup to avoid N+1 queries.
	groupUIDs := make([]string, 0, len(rules))
	seen := map[string]struct{}{}
	for _, r := range rules {
		if _, ok := seen[r.GroupUID]; !ok && r.GroupUID != "" {
			groupUIDs = append(groupUIDs, r.GroupUID)
			seen[r.GroupUID] = struct{}{}
		}
	}

	if len(groupUIDs) > 0 {
		var groups []model.AlertRuleGroup
		placeholders := strings.Repeat("?,", len(groupUIDs))
		placeholders = placeholders[:len(placeholders)-1] // trim trailing comma
		inArgs := make([]any, 0, len(groupUIDs)+1)
		inArgs = append(inArgs, "uid in ("+placeholders+")")
		for _, uid := range groupUIDs {
			inArgs = append(inArgs, uid)
		}
		if err := srv.db.Find(&groups, inArgs...); err != nil {
			return nil, err
		}

		intervalByUID := make(map[string]time.Duration, len(groups))
		for _, g := range groups {
			intervalByUID[g.UID] = g.Interval
		}
		// Back-fill group interval into each rule so the scheduler needs zero changes.
		for i := range rules {
			if iv, ok := intervalByUID[rules[i].GroupUID]; ok {
				rules[i].Interval = iv
			}
		}
	}
	return rules, nil
}
