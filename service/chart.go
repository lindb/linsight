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

//go:generate mockgen -source=./chart.go -destination=./chart_mock.go -package=service

// ChartService represents chart repo manager interface.
type ChartService interface {
	// SearchCharts searches the charts by given params.
	SearchCharts(ctx context.Context, req *model.SearchChartRequest) (rs []model.ChartInfo, total int64, err error)
	// CreateChart creates a chart.
	CreateChart(ctx context.Context, chart *model.Chart) (string, error)
	// UpdateChart updates the chart by uid.
	UpdateChart(ctx context.Context, chart *model.Chart) error
	// DeleteChartByUID deletes the chart by uid.
	DeleteChartByUID(ctx context.Context, uid string) error
	// GetChartByUID returns the chart by uid.
	GetChartByUID(ctx context.Context, uid string) (*model.Chart, error)
	// LinkChartsToDashboard links charts to specific dashboard.
	LinkChartsToDashboard(ctx context.Context, dashboard *model.Dashboard) error
	// UnlinkChartFromDashboard unlinks chart from specific dashboard.
	UnlinkChartFromDashboard(ctx context.Context, chartUID, dashboardUID string) error
}

// chartService implements ChartService interface.
type chartService struct {
	db dbpkg.DB
}

// NewChartService creates a ChartService instance.
func NewChartService(db dbpkg.DB) ChartService {
	return &chartService{
		db: db,
	}
}

// CreateChart creates a chart.
func (srv *chartService) CreateChart(ctx context.Context, chart *model.Chart) (string, error) {
	chart.UID = uuid.GenerateShortUUID()
	// set chart org/user info
	user := util.GetUser(ctx)
	chart.OrgID = user.Org.ID
	userID := user.User.ID
	chart.CreatedBy = userID
	chart.UpdatedBy = userID
	if err := srv.db.Create(chart); err != nil {
		return "", err
	}
	return chart.UID, nil
}

// UpdateChart updates the chart by uid.
func (srv *chartService) UpdateChart(ctx context.Context, chart *model.Chart) error {
	chartFromDB, err := srv.GetChartByUID(ctx, chart.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	// update chart
	chartFromDB.Title = chart.Title
	chartFromDB.Desc = chart.Desc
	chartFromDB.Model = chart.Model
	chartFromDB.UpdatedBy = user.User.ID
	return srv.db.Update(chartFromDB, "uid=? and org_id=?", chart.UID, user.Org.ID)
}

// SearchCharts searches the chart by given params.
func (srv *chartService) SearchCharts(ctx context.Context,
	req *model.SearchChartRequest,
) (rs []model.ChartInfo, total int64, err error) {
	conditions := []string{"org_id=?"}
	signedUser := util.GetUser(ctx)
	params := []any{signedUser.Org.ID}
	sql := `select c.uid,c.title,c.desc,
		(select count(1) from links l where l.source_uid=c.uid) as dashboards  
	from charts c where c.org_id=?`
	if req.Title != "" {
		conditions = append(conditions, "title like ?")
		params = append(params, req.Title+"%")
		sql += " title like ? "
	}
	if req.Ownership == model.Mine {
		conditions = append(conditions, "created_by=?")
		params = append(params, signedUser.User.ID)
		sql += " created_by=? "
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.Chart{}, where, params...)
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
	sql += " order by c.id desc limit ? offset ?"
	params = append(params, limit, offset)
	if err := srv.db.ExecRaw(&rs, sql, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// DeleteChartByUID deletes the chart by uid.
func (srv *chartService) DeleteChartByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// delete chart
		return tx.Delete(&model.Chart{}, "uid=? and org_id=?", uid, orgID)
	})
}

// GetChartByUID returns the chart by uid.
func (srv *chartService) GetChartByUID(ctx context.Context, uid string) (*model.Chart, error) {
	rs := &model.Chart{}
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

// LinkChartsToDashboard links charts to specific dashboard.
func (srv *chartService) LinkChartsToDashboard(ctx context.Context, dashboard *model.Dashboard) error {
	chartUIDs, err := dashboard.GetCharts()
	if err != nil {
		return err
	}
	signedUser := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		dashboardUID := dashboard.UID
		// 1. delete current chart links
		if err := tx.Delete(&model.Link{},
			"org_id=? and kind=? and target_uid=?",
			signedUser.Org.ID, model.DashboardLink, dashboardUID); err != nil {
			return err
		}
		// 2. add new links
		for _, chartUID := range chartUIDs {
			// create new link
			if err := tx.Create(&model.Link{
				OrgID:     signedUser.Org.ID,
				SourceUID: chartUID,
				TargetUID: dashboardUID,
				Kind:      model.DashboardLink,
				BaseModel: model.BaseModel{
					CreatedBy: signedUser.User.ID,
					UpdatedBy: signedUser.User.ID,
				},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// UnlinkChartFromDashboard unlinks chart from specific dashboard.
func (srv *chartService) UnlinkChartFromDashboard(ctx context.Context, chartUID, dashboardUID string) error {
	signedUser := util.GetUser(ctx)
	// delete current chart links
	return srv.db.Delete(&model.Link{},
		"org_id=? and kind=? and source_uid=? and target_uid=?",
		signedUser.Org.ID, model.DashboardLink, chartUID, dashboardUID)
}
