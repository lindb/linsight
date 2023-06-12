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
)

//go:generate mockgen -source=./nav.go -destination=./nav_mock.go -package=service

// NavService represents navigation manager interface.
type NavService interface {
	// GetNavByOrgID returns the navigation by given org id.
	GetNavByOrgID(ctx context.Context, orgID int64) (*model.Nav, error)
	// UpdateNav updates the navigation.
	UpdateNav(ctx context.Context, orgIDd int64, nav *model.Nav) error
}

// navService implements NavService intreface.
type navService struct {
	db dbpkg.DB
}

// NewNavService creates a NavService instance.
func NewNavService(db dbpkg.DB) NavService {
	return &navService{
		db: db,
	}
}

// GetNavByOrgID returns the navigation by given org id.
func (srv *navService) GetNavByOrgID(ctx context.Context, orgID int64) (*model.Nav, error) {
	rs := &model.Nav{}
	if err := srv.db.Get(rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return rs, nil
}

// UpdateNav updates the navigation.
func (srv *navService) UpdateNav(ctx context.Context, orgID int64, nav *model.Nav) error {
	navFromDB, err := srv.GetNavByOrgID(ctx, orgID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	navFromDB.UpdatedBy = user.User.ID
	navFromDB.Config = nav.Config
	return srv.db.Update(navFromDB, "org_id=?", user.Org.ID)
}
