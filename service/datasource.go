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

//go:generate mockgen -source=./datasource.go -destination=./datasource_mock.go -package=service

// DatasourceService represents data source manager interface.
type DatasourceService interface {
	// CreateDatasource creates a data source, if success returns uid of data source.
	CreateDatasource(ctx context.Context, datasource *model.Datasource) (uid string, err error)
	// UpdateDatasource updates data source.
	UpdateDatasource(ctx context.Context, datasource *model.Datasource) error
	// DeleteDatasourceByUID deletes data source by uid from current org.
	DeleteDatasourceByUID(ctx context.Context, uid string) error
	// GetDatasources returns all data sources for current org.
	GetDatasources(ctx context.Context) ([]model.Datasource, error)
	// GetDatasourceByUID returns data source by uid from current org.
	GetDatasourceByUID(ctx context.Context, uid string) (*model.Datasource, error)
	GetDatasourceByOrgAndUID(orgID int64, uid string) (*model.Datasource, error)
}

// datasourceService implements DatasourceService interface.
type datasourceService struct {
	db dbpkg.DB
}

// NewDatasourceService creates a DatasourceService instance.
func NewDatasourceService(db dbpkg.DB) DatasourceService {
	return &datasourceService{
		db: db,
	}
}

// CreateDatasource creates a data source, if success returns uid of data source.
func (srv *datasourceService) CreateDatasource(ctx context.Context, datasource *model.Datasource) (uid string, err error) {
	datasource.UID = uuid.GenerateShortUUID()
	err = srv.db.Transaction(func(tx dbpkg.DB) error {
		user := util.GetUser(ctx)
		if err0 := srv.cleanDefaultDatasource(tx, user.Org.ID, datasource); err0 != nil {
			return err0
		}

		datasource.OrgID = user.Org.ID
		userID := user.User.ID
		datasource.CreatedBy = userID
		datasource.UpdatedBy = userID

		return tx.Create(&datasource)
	})
	if err != nil {
		return "", err
	}
	return datasource.UID, nil
}

// UpdateDatasource updates data source.
func (srv *datasourceService) UpdateDatasource(ctx context.Context, datasource *model.Datasource) error {
	ds, err := srv.GetDatasourceByUID(ctx, datasource.UID)
	if err != nil {
		return err
	}
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		user := util.GetUser(ctx)
		userID := user.User.ID
		ds.UpdatedBy = userID

		if err := srv.cleanDefaultDatasource(tx, user.Org.ID, datasource); err != nil {
			return err
		}

		// update datasource
		ds.URL = datasource.URL
		ds.Config = datasource.Config
		if err := tx.Updates(&model.Datasource{},
			map[string]any{
				"name":       datasource.Name,
				"is_default": datasource.IsDefault,
				"url":        datasource.URL,
				"config":     datasource.Config,
				"time_zone":  datasource.TimeZone,
				"updated_by": userID,
			}, "uid=? and org_id=?", datasource.UID, user.Org.ID); err != nil {
			return err
		}
		return nil
	})
}

// DeleteDatasourceByUID deletes data source by uid from current org.
func (srv *datasourceService) DeleteDatasourceByUID(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	return srv.db.Delete(&model.Datasource{}, "uid=? and org_id=?", uid, user.Org.ID)
}

// GetDatasources returns all data sources for current org.
func (srv *datasourceService) GetDatasources(ctx context.Context) ([]model.Datasource, error) {
	var rs []model.Datasource

	user := util.GetUser(ctx)
	if err := srv.db.Find(&rs, "org_id=?", user.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

// GetDatasourceByUID returns data source by uid from current org.
func (srv *datasourceService) GetDatasourceByUID(ctx context.Context, uid string) (*model.Datasource, error) {
	user := util.GetUser(ctx)
	return srv.GetDatasourceByOrgAndUID(user.User.OrgID, uid)
}

// cleanDefaultDatasource cleans default datasource if exist when set new default datasource.
func (srv *datasourceService) cleanDefaultDatasource(tx dbpkg.DB, orgID int64, datasource *model.Datasource) error {
	if datasource.IsDefault {
		// update old default datasource
		return tx.UpdateSingle(&model.Datasource{}, "is_default", false, "org_id=?", orgID)
	}
	return nil
}

func (srv *datasourceService) GetDatasourceByOrgAndUID(orgID int64, uid string) (rs *model.Datasource, err error) {
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
