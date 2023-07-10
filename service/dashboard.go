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

//go:generate mockgen -source=./dashboard.go -destination=./dashboard_mock.go -package=service

// DashboardService represents dashboard manager interface.
type DashboardService interface {
	// CreateDashboard creates a dashboard.
	CreateDashboard(ctx context.Context, dashboard *model.Dashboard) (string, error)
	// UpdateDashboard updates the dashboard by uid.
	UpdateDashboard(ctx context.Context, dashboard *model.Dashboard) error
	// DeleteDashboardByUID deletes the dashboard by uid.
	DeleteDashboardByUID(ctx context.Context, uid string) error
	// SearchDashboards searches the dashboard by given params.
	SearchDashboards(ctx context.Context, req *model.SearchDashboardRequest) (rs []model.Dashboard, total int64, err error)
	// GetDashboardByUID returns the dashboard by uid.
	GetDashboardByUID(ctx context.Context, uid string) (*model.Dashboard, error)
	// StarDashboard stars the dashboard by uid.
	StarDashboard(ctx context.Context, uid string) error
	// UnstarDashboard unstars the dashboard by uid.
	UnstarDashboard(ctx context.Context, uid string) error
	// GetDashboardsByChartUID returns dashboards by chart.
	GetDashboardsByChartUID(ctx context.Context, chartUID string) (rs []model.Dashboard, err error)
}

// dashboardService implements DashboardService interface.
type dashboardService struct {
	db      dbpkg.DB
	starSrv StarService
}

// NewDashboardService creates a DashboardService instance.
func NewDashboardService(starSrv StarService, db dbpkg.DB) DashboardService {
	return &dashboardService{
		db:      db,
		starSrv: starSrv,
	}
}

// CreateDashboard creates a dashboard.
func (srv *dashboardService) CreateDashboard(ctx context.Context, dashboard *model.Dashboard) (string, error) {
	dashboard.UID = uuid.GenerateShortUUID()
	// set dashboard org/user info
	user := util.GetUser(ctx)
	dashboard.OrgID = user.Org.ID
	userID := user.User.ID
	dashboard.CreatedBy = userID
	dashboard.UpdatedBy = userID
	if err := srv.db.Create(dashboard); err != nil {
		return "", err
	}
	return dashboard.UID, nil
}

// UpdateDashboard updates the dashboard by uid.
func (srv *dashboardService) UpdateDashboard(ctx context.Context, dashboard *model.Dashboard) error {
	dashboardFromDB, err := srv.getDashboardByUID(ctx, dashboard.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	userID := user.User.ID

	// update datasource
	dashboardFromDB.Title = dashboard.Title
	dashboardFromDB.Desc = dashboard.Desc
	dashboardFromDB.Config = dashboard.Config
	dashboardFromDB.UpdatedBy = userID
	return srv.db.Update(dashboardFromDB, "uid=? and org_id=?", dashboard.UID, user.Org.ID)
}

// DeleteDashboardByUID deletes the dashboard by uid.
func (srv *dashboardService) DeleteDashboardByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// delete dashboard
		if err := tx.Delete(&model.Dashboard{}, "uid=? and org_id=?", uid, orgID); err != nil {
			return err
		}
		// delete start record
		return tx.Delete(&model.Star{}, "org_id=? and entity_id=? and entity_type=?", orgID, uid, model.DasbboardEntity)
	})
}

// SearchDashboards searches the dashboard by given params.
func (srv *dashboardService) SearchDashboards(ctx context.Context,
	req *model.SearchDashboardRequest,
) (rs []model.Dashboard, total int64, err error) {
	conditions := []string{"org_id=?"}
	signedUser := util.GetUser(ctx)
	params := []any{signedUser.Org.ID}
	if req.Title != "" {
		conditions = append(conditions, "title like ?")
		params = append(params, req.Title+"%")
	}
	if req.Ownership == model.Mine {
		conditions = append(conditions, "created_by=?")
		params = append(params, signedUser.User.ID)
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
	// TODO: refactor set select columns
	count, err := srv.db.Count(&model.Dashboard{}, where, params...)
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

// GetDashboardByUID returns the dashboard by uid.
func (srv *dashboardService) GetDashboardByUID(ctx context.Context, uid string) (*model.Dashboard, error) {
	rs, err := srv.getDashboardByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	isStarred, err := srv.starSrv.IsStarred(ctx, rs.ID, model.DasbboardEntity)
	if err != nil {
		//TODO: ignore check star err:
		return nil, err
	}
	rs.IsStarred = isStarred
	return rs, nil
}

// StarDashboard stars the dashboard by uid.
func (srv *dashboardService) StarDashboard(ctx context.Context, uid string) error {
	return srv.toggleStar(ctx, uid, true)
}

// UnstarDashboard unstars the dashboard by uid.
func (srv *dashboardService) UnstarDashboard(ctx context.Context, uid string) error {
	return srv.toggleStar(ctx, uid, false)
}

// GetDashboardsByChartUID returns dashboards by chart.
func (srv *dashboardService) GetDashboardsByChartUID(ctx context.Context, chartUID string) (rs []model.Dashboard, err error) {
	sql := `select d.uid,d.title,d.desc from dashboards d,links l where d.uid=l.target_uid and d.org_id=? and l.source_uid=?`
	signedUser := util.GetUser(ctx)
	if err := srv.db.ExecRaw(&rs, sql, signedUser.Org.ID, chartUID); err != nil {
		return nil, err
	}
	return
}

// toggleStar toggles the dashboard star.
func (srv *dashboardService) toggleStar(ctx context.Context, uid string, star bool) error {
	dashboard, err := srv.getDashboardByUID(ctx, uid)
	if err != nil {
		return err
	}
	if star {
		return srv.starSrv.Star(ctx, dashboard.ID, model.DasbboardEntity)
	} else {
		return srv.starSrv.Unstar(ctx, dashboard.ID, model.DasbboardEntity)
	}
}

// getDashboardByUID returns the dashboard by uid.
func (srv *dashboardService) getDashboardByUID(ctx context.Context, uid string) (*model.Dashboard, error) {
	rs := &model.Dashboard{}
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}
