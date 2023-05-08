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
	// CreateChart creates a chart.
	CreateChart(ctx context.Context, chart *model.Chart) (string, error)
	// SearchCharts searches the charts by given params.
	SearchCharts(ctx context.Context, req *model.SearchChartRequest) (rs []model.Chart, total int64, err error)
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

// SearchCharts searches the chart by given params.
func (srv *chartService) SearchCharts(ctx context.Context, //nolint:dupl
	req *model.SearchChartRequest,
) (rs []model.Chart, total int64, err error) {
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
	count, err := srv.db.Count(&model.Chart{}, where, params...)
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
