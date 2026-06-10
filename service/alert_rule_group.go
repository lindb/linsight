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

//go:generate mockgen -source=./alert_rule_group.go -destination=./alert_rule_group_mock.go -package=service

import (
	"context"
	"fmt"
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// AlertRuleGroupService manages alert rule groups.
// Groups are created implicitly via CreateAlertRule (find-or-create by name); no standalone create endpoint exists.
type AlertRuleGroupService interface {
	// FindOrCreateAlertRuleGroup looks up a group by (orgID, name) within a transaction.
	// If not found it creates a new group using param.Interval; returns error if the group does not exist
	// and param.Interval is zero.
	FindOrCreateAlertRuleGroup(tx dbpkg.DB, orgID, userID int64, param *model.CreateAlertRuleGroupParam) (*model.AlertRuleGroup, error)
	// UpdateAlertRuleGroup updates the mutable fields of a group.
	UpdateAlertRuleGroup(ctx context.Context, group *model.AlertRuleGroup) error
	// DeleteAlertRuleGroupByUID deletes a group; returns an error if the group still has rules.
	DeleteAlertRuleGroupByUID(ctx context.Context, uid string) error
	// GetAlertRuleGroupByUID returns a group by its UID.
	GetAlertRuleGroupByUID(ctx context.Context, uid string) (*model.AlertRuleGroup, error)
	// SearchAlertRuleGroups searches groups by given params, scoped to the signed-in user's org.
	SearchAlertRuleGroups(ctx context.Context, req *model.SearchAlertRuleGroupRequest) ([]model.AlertRuleGroup, int64, error)
}

type alertRuleGroupService struct {
	db dbpkg.DB
}

// NewAlertRuleGroupService creates an AlertRuleGroupService instance.
func NewAlertRuleGroupService(db dbpkg.DB) AlertRuleGroupService {
	return &alertRuleGroupService{db: db}
}

// FindOrCreateAlertRuleGroup looks up or creates a group within an existing transaction.
// Caller is responsible for wrapping this in a transaction.
func (srv *alertRuleGroupService) FindOrCreateAlertRuleGroup(tx dbpkg.DB, orgID, userID int64, param *model.CreateAlertRuleGroupParam) (*model.AlertRuleGroup, error) {
	var group model.AlertRuleGroup
	err := tx.Get(&group, "org_id=? and name=?", orgID, param.Name)
	if err == nil {
		// Group already exists; reuse it.
		return &group, nil
	}

	// Group not found — create it. Interval must be provided for new groups.
	if param.Interval <= 0 {
		return nil, fmt.Errorf("interval is required when creating a new alert rule group %q", param.Name)
	}
	group = model.AlertRuleGroup{
		OrgID:    orgID,
		UID:      uuid.GenerateShortUUID(),
		Name:     param.Name,
		Interval: param.Interval,
	}
	group.CreatedBy = userID
	group.UpdatedBy = userID
	if err := tx.Create(&group); err != nil {
		return nil, fmt.Errorf("create alert rule group %q: %w", param.Name, err)
	}
	return &group, nil
}

// UpdateAlertRuleGroup updates description and interval for the given group (scoped to the signed-in org).
func (srv *alertRuleGroupService) UpdateAlertRuleGroup(ctx context.Context, group *model.AlertRuleGroup) error {
	existing, err := srv.GetAlertRuleGroupByUID(ctx, group.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	existing.Name = group.Name
	existing.Interval = group.Interval
	existing.UpdatedBy = user.User.ID
	return srv.db.Update(existing, "uid=? and org_id=?", group.UID, user.Org.ID)
}

// DeleteAlertRuleGroupByUID deletes a group only if it has no rules.
func (srv *alertRuleGroupService) DeleteAlertRuleGroupByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID

	// Guard: refuse deletion when the group still owns rules.
	count, err := srv.db.Count(&model.AlertRule{}, "group_uid=?", uid)
	if err != nil {
		return fmt.Errorf("check rules in group %q: %w", uid, err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete group: it has %d rule(s); please move or delete them first", count)
	}
	return srv.db.Delete(&model.AlertRuleGroup{}, "uid=? and org_id=?", uid, orgID)
}

// GetAlertRuleGroupByUID returns the group scoped to the signed-in user's org.
func (srv *alertRuleGroupService) GetAlertRuleGroupByUID(ctx context.Context, uid string) (*model.AlertRuleGroup, error) {
	signedUser := util.GetUser(ctx)
	var group model.AlertRuleGroup
	if err := srv.db.Get(&group, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return &group, nil
}

// SearchAlertRuleGroups searches groups by name prefix, paginated.
func (srv *alertRuleGroupService) SearchAlertRuleGroups(ctx context.Context, req *model.SearchAlertRuleGroupRequest) ([]model.AlertRuleGroup, int64, error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")

	count, err := srv.db.Count(&model.AlertRuleGroup{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}

	offset, limit := 0, 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}

	var rs []model.AlertRuleGroup
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}
