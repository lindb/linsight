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

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

type OrgService interface {
	CreateOrg(ctx context.Context, org *model.Org) (string, error)
	UpdateOrg(ctx context.Context, org *model.Org) error
	DeleteOrg(ctx context.Context, uid string) error
	GetOrgByUID(ctx context.Context, uid string) (*model.Org, error)
	SearchOrg(ctx context.Context) ([]model.Org, error)
}

type orgService struct {
	db dbpkg.DB
}

func NewOrgService(db dbpkg.DB) OrgService {
	return &orgService{
		db: db,
	}
}

func (srv *orgService) CreateOrg(ctx context.Context, org *model.Org) (string, error) {
	uid := uuid.GenerateShortUUID()
	signedUser := util.GetUser(ctx)
	org.UID = uid
	org.CreatedBy = signedUser.User.ID
	org.UpdatedBy = signedUser.User.ID
	if err := srv.db.Create(org); err != nil {
		return "", err
	}
	return uid, nil
}

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

func (srv *orgService) DeleteOrg(ctx context.Context, uid string) error {
	// FIXME: delete all resources
	return srv.db.Delete(&model.Org{}, "uid=?", uid)
}

func (srv *orgService) GetOrgByUID(ctx context.Context, uid string) (*model.Org, error) {
	org := &model.Org{}
	if err := srv.db.Get(org, "uid=?", uid); err != nil {
		return nil, err
	}
	return org, nil
}

func (srv *orgService) SearchOrg(ctx context.Context) ([]model.Org, error) {
	var rs []model.Org
	if err := srv.db.Find(rs); err != nil {
		return nil, err
	}
	return rs, nil
}
