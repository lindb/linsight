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

	"gorm.io/datatypes"

	"github.com/lindb/linsight"
	"github.com/lindb/linsight/accesscontrol"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

//go:generate mockgen -source=./org.go -destination=./org_mock.go -package=service

// OrgService represents organization manager interface.
type OrgService interface {
	// CreateOrg creates a new organization, returns organization uid, if fail returns error.
	CreateOrg(ctx context.Context, org *model.Org) (string, error)
	// UpdateOrg updates the organization basic information.
	UpdateOrg(ctx context.Context, org *model.Org) error
	// DeleteOrgByUID deletes the organization by given uid.
	DeleteOrgByUID(ctx context.Context, uid string) error
	// GetOrgByUID returns the organization by given uid.
	GetOrgByUID(ctx context.Context, uid string) (*model.Org, error)
	// SearchOrg searches the organization by given params.
	SearchOrg(ctx context.Context, req *model.SearchOrgRequest) ([]model.Org, int64, error)
	// GetOrgListForSignedUser returns all org for current signed user can manage.
	GetOrgListForSignedUser(ctx context.Context) ([]model.Org, error)
}

// orgService implements OrgService interface.
type orgService struct {
	db dbpkg.DB
}

// NewOrgService create an OrgService instance.
func NewOrgService(db dbpkg.DB) OrgService {
	return &orgService{
		db: db,
	}
}

// CreateOrg creates a new organization, returns organization uid, if fail returns error.
func (srv *orgService) CreateOrg(ctx context.Context, org *model.Org) (string, error) {
	uid := uuid.GenerateShortUUID()
	err := srv.db.Transaction(func(tx dbpkg.DB) error {
		signedUser := util.GetUser(ctx)
		org.UID = uid
		org.CreatedBy = signedUser.User.ID
		org.UpdatedBy = signedUser.User.ID
		if err := tx.Create(org); err != nil {
			return err
		}
		// add nav tree for new org
		nav := &model.Nav{
			OrgID:         org.ID,
			Config:        datatypes.JSON(linsight.DefaultNav),
			DefaultConfig: datatypes.JSON(linsight.DefaultNav),
		}
		nav.CreatedBy = signedUser.User.ID
		nav.UpdatedBy = signedUser.User.ID
		return tx.Create(nav)
	})
	if err != nil {
		return "", err
	}
	return uid, nil
}

// UpdateOrg updates the organization basic information.
func (srv *orgService) UpdateOrg(ctx context.Context, org *model.Org) error {
	orgFromDB, err := srv.GetOrgByUID(ctx, org.UID)
	if err != nil {
		return err
	}
	signedUser := util.GetUser(ctx)
	orgFromDB.UpdatedBy = signedUser.User.ID
	orgFromDB.Name = org.Name
	return srv.db.Update(orgFromDB, "uid=?", org.UID)
}

// DeleteOrgByUID deletes the organization by given uid.
func (srv *orgService) DeleteOrgByUID(ctx context.Context, uid string) error {
	// FIXME: delete all resources
	return srv.db.Delete(&model.Org{}, "uid=?", uid)
}

// GetOrgByUID returns the organization by given uid.
func (srv *orgService) GetOrgByUID(ctx context.Context, uid string) (*model.Org, error) {
	org := &model.Org{}
	if err := srv.db.Get(org, "uid=?", uid); err != nil {
		return nil, err
	}
	return org, nil
}

// SearchOrg searches the organization by given params.
func (srv *orgService) SearchOrg(ctx context.Context, req *model.SearchOrgRequest) ([]model.Org, int64, error) {
	var rs []model.Org
	conditions := []string{}
	params := []any{}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.Org{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// GetOrgListForSignedUser returns all org for current signed user can manage.
func (srv *orgService) GetOrgListForSignedUser(ctx context.Context) (rs []model.Org, err error) {
	conditions := []string{}
	params := []any{}
	signedUser := util.GetUser(ctx)
	// if role is Lin can return all orgs, else return orgs which has admin role
	if signedUser.Role != accesscontrol.RoleLin {
		conditions = append(conditions, "id in (select ou.org_id from org_users ou where ou.user_id=?)")
		params = append(params, signedUser.User.ID)
		conditions = append(conditions, "role=?")
		params = append(params, accesscontrol.RoleAdmin)
	}
	findParams := []any{strings.Join(conditions, " and ")}
	findParams = append(findParams, params...)
	if err := srv.db.Find(&rs, findParams...); err != nil {
		return nil, err
	}
	return rs, nil
}
