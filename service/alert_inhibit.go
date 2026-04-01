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
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// AlertInhibitRuleService manages alert inhibit rules.
type AlertInhibitRuleService interface {
	// CreateAlertInhibitRule creates a new inhibit rule.
	CreateAlertInhibitRule(ctx context.Context, rule *model.AlertInhibitRule) (string, error)
	// UpdateAlertInhibitRule updates an existing inhibit rule.
	UpdateAlertInhibitRule(ctx context.Context, rule *model.AlertInhibitRule) error
	// DeleteAlertInhibitRuleByUID removes an inhibit rule.
	DeleteAlertInhibitRuleByUID(ctx context.Context, uid string) error
	// GetAlertInhibitRuleByUID returns an inhibit rule by UID.
	GetAlertInhibitRuleByUID(ctx context.Context, uid string) (*model.AlertInhibitRule, error)
	// SearchAlertInhibitRules queries inhibit rules with optional filters.
	SearchAlertInhibitRules(ctx context.Context, req *model.SearchAlertInhibitRuleRequest) (rs []model.AlertInhibitRule, total int64, err error)
	// GetEnabledInhibitRulesByOrg returns all enabled inhibit rules for an org (used by the alert engine).
	GetEnabledInhibitRulesByOrg(orgID int64) ([]*model.AlertInhibitRule, error)
}

type alertInhibitRuleService struct {
	db dbpkg.DB
}

// NewAlertInhibitRuleService creates an AlertInhibitRuleService instance.
func NewAlertInhibitRuleService(db dbpkg.DB) AlertInhibitRuleService {
	return &alertInhibitRuleService{db: db}
}

func (srv *alertInhibitRuleService) CreateAlertInhibitRule(ctx context.Context, rule *model.AlertInhibitRule) (string, error) {
	rule.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	rule.OrgID = user.Org.ID
	rule.CreatedBy = user.User.ID
	rule.UpdatedBy = user.User.ID
	if err := srv.db.Create(rule); err != nil {
		return "", err
	}
	return rule.UID, nil
}

func (srv *alertInhibitRuleService) UpdateAlertInhibitRule(ctx context.Context, rule *model.AlertInhibitRule) error {
	fromDB, err := srv.GetAlertInhibitRuleByUID(ctx, rule.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = rule.Name
	fromDB.SourceMatchers = rule.SourceMatchers
	fromDB.TargetMatchers = rule.TargetMatchers
	fromDB.EqualLabels = rule.EqualLabels
	fromDB.Enabled = rule.Enabled
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", rule.UID, user.Org.ID)
}

func (srv *alertInhibitRuleService) DeleteAlertInhibitRuleByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.AlertInhibitRule{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *alertInhibitRuleService) GetAlertInhibitRuleByUID(ctx context.Context, uid string) (rs *model.AlertInhibitRule, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertInhibitRuleService) SearchAlertInhibitRules(ctx context.Context, req *model.SearchAlertInhibitRuleRequest) (rs []model.AlertInhibitRule, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertInhibitRule{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

func (srv *alertInhibitRuleService) GetEnabledInhibitRulesByOrg(orgID int64) ([]*model.AlertInhibitRule, error) {
	var rs []*model.AlertInhibitRule
	if err := srv.db.Find(&rs, "org_id=? and enabled=?", orgID, true); err != nil {
		return nil, err
	}
	return rs, nil
}
