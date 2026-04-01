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

// AlertRouteService manages alert routing tree nodes.
type AlertRouteService interface {
	// GetAlertRouteTree returns all route nodes for the current org.
	GetAlertRouteTree(ctx context.Context) ([]*model.AlertRoute, error)
	// CreateAlertRoute creates a route node.
	CreateAlertRoute(ctx context.Context, route *model.AlertRoute) (string, error)
	// UpdateAlertRoute updates a route node.
	UpdateAlertRoute(ctx context.Context, route *model.AlertRoute) error
	// DeleteAlertRouteByUID removes a route node.
	DeleteAlertRouteByUID(ctx context.Context, uid string) error
	// GetAlertRouteByUID returns a single route node.
	GetAlertRouteByUID(ctx context.Context, uid string) (*model.AlertRoute, error)
	// SearchAlertRoutes searches routes by name.
	SearchAlertRoutes(ctx context.Context, req *model.SearchAlertRouteRequest) (rs []model.AlertRoute, total int64, err error)
	// GetAlertRoutesByOrg returns all routes for an org (used by the alert engine).
	GetAlertRoutesByOrg(orgID int64) ([]*model.AlertRoute, error)
}

type alertRouteService struct {
	db dbpkg.DB
}

// NewAlertRouteService creates an AlertRouteService instance.
func NewAlertRouteService(db dbpkg.DB) AlertRouteService {
	return &alertRouteService{db: db}
}

func (srv *alertRouteService) GetAlertRouteTree(ctx context.Context) ([]*model.AlertRoute, error) {
	signedUser := util.GetUser(ctx)
	return srv.GetAlertRoutesByOrg(signedUser.Org.ID)
}

func (srv *alertRouteService) CreateAlertRoute(ctx context.Context, route *model.AlertRoute) (string, error) {
	route.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	route.OrgID = user.Org.ID
	route.CreatedBy = user.User.ID
	route.UpdatedBy = user.User.ID
	if err := srv.db.Create(route); err != nil {
		return "", err
	}
	return route.UID, nil
}

func (srv *alertRouteService) UpdateAlertRoute(ctx context.Context, route *model.AlertRoute) error {
	fromDB, err := srv.GetAlertRouteByUID(ctx, route.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = route.Name
	fromDB.ParentUID = route.ParentUID
	fromDB.Matchers = route.Matchers
	fromDB.ReceiverUID = route.ReceiverUID
	fromDB.Continue = route.Continue
	fromDB.GroupBy = route.GroupBy
	fromDB.GroupWait = route.GroupWait
	fromDB.GroupInterval = route.GroupInterval
	fromDB.RepeatInterval = route.RepeatInterval
	fromDB.SortOrder = route.SortOrder
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", route.UID, user.Org.ID)
}

func (srv *alertRouteService) DeleteAlertRouteByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.AlertRoute{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *alertRouteService) GetAlertRouteByUID(ctx context.Context, uid string) (rs *model.AlertRoute, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertRouteService) SearchAlertRoutes(ctx context.Context, req *model.SearchAlertRouteRequest) (rs []model.AlertRoute, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertRoute{}, where, params...)
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
	if err := srv.db.FindForPaging(&rs, offset, limit, "sort_order asc, id asc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

func (srv *alertRouteService) GetAlertRoutesByOrg(orgID int64) ([]*model.AlertRoute, error) {
	var rs []*model.AlertRoute
	if err := srv.db.Find(&rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
