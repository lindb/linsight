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
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

// ─── buildRouteTree (pure function) ──────────────────────────────────────────

func TestBuildRouteTree_EmptySlice(t *testing.T) {
	assert.Nil(t, buildRouteTree(nil))
	assert.Nil(t, buildRouteTree([]*model.AlertRoute{}))
}

func TestBuildRouteTree_NoDefault(t *testing.T) {
	// Routes exist but none has ParentUID="" → tree cannot be built.
	routes := []*model.AlertRoute{
		{UID: "r1", ParentUID: "missing-root"},
		{UID: "r2", ParentUID: "r1"},
	}
	assert.Nil(t, buildRouteTree(routes))
}

func TestBuildRouteTree_SingleDefault(t *testing.T) {
	root := &model.AlertRoute{UID: "root", ParentUID: "", Name: "catch-all"}
	node := buildRouteTree([]*model.AlertRoute{root})

	assert.NotNil(t, node)
	assert.Equal(t, "root", node.UID)
	assert.Empty(t, node.Routes)
}

func TestBuildRouteTree_NestedTree(t *testing.T) {
	// Tree layout:
	//   root (ParentUID="" → root marker)
	//   ├── child1 (SortOrder=1)
	//   │   └── grandchild
	//   └── child2 (SortOrder=2)
	routes := []*model.AlertRoute{
		{BaseModel: model.BaseModel{ID: 1}, UID: "root", ParentUID: ""},
		{BaseModel: model.BaseModel{ID: 3}, UID: "child2", ParentUID: "root", SortOrder: 2},
		{BaseModel: model.BaseModel{ID: 2}, UID: "child1", ParentUID: "root", SortOrder: 1},
		{BaseModel: model.BaseModel{ID: 4}, UID: "grandchild", ParentUID: "child1"},
	}

	node := buildRouteTree(routes)

	assert.NotNil(t, node)
	assert.Equal(t, "root", node.UID)
	// Children should be sorted: child1 (SortOrder=1) before child2 (SortOrder=2)
	assert.Len(t, node.Routes, 2)
	assert.Equal(t, "child1", node.Routes[0].UID)
	assert.Equal(t, "child2", node.Routes[1].UID)
	// Grandchild is nested under child1
	assert.Len(t, node.Routes[0].Routes, 1)
	assert.Equal(t, "grandchild", node.Routes[0].Routes[0].UID)
}

// ─── CreateAlertRoute ────────────────────────────────────────────────────────

func TestCreateAlertRoute_NoParentUID_ReturnsError(t *testing.T) {
	// Root route (ParentUID="") cannot be created via API — it is created automatically
	// when the org is initialized. The service must reject such requests immediately.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	route := &model.AlertRoute{Name: "default policy", ParentUID: ""}

	// No DB calls expected — the validation fails before touching the database.
	_, err := srv.CreateAlertRoute(ctx, &model.CreateAlertRouteRequest{AlertRoute: *route})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parentUID is required")
}

func TestCreateAlertRoute_WithParentUID_Success(t *testing.T) {
	// A route with a valid ParentUID pointing to an existing parent route is created normally.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	route := &model.AlertRoute{Name: "specific policy", ParentUID: "root-uid"}

	// no contact point validation (ContactPointUID is empty)
	// parent route exists
	mockDB.EXPECT().Exist(gomock.Any(), "org_id=? and uid=?", int64(12), "root-uid").Return(true, nil)
	mockDB.EXPECT().Create(gomock.Any()).Return(nil)

	uid, err := srv.CreateAlertRoute(ctx, &model.CreateAlertRouteRequest{AlertRoute: *route})
	assert.NoError(t, err)
	assert.NotEmpty(t, uid)
}

func TestCreateAlertRoute_ParentNotFound_ReturnsError(t *testing.T) {
	// When the specified parent route does not exist, the service returns an error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	route := &model.AlertRoute{Name: "child policy", ParentUID: "nonexistent-uid"}

	mockDB.EXPECT().Exist(gomock.Any(), "org_id=? and uid=?", int64(12), "nonexistent-uid").Return(false, nil)

	_, err := srv.CreateAlertRoute(ctx, &model.CreateAlertRouteRequest{AlertRoute: *route})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent route")
}

// ─── CreateAlertRoute sibling insertion ──────────────────────────────────────
// Note: happy-path sibling insertion tests (before/after) are covered by integration tests
// because the shift step uses RawDB() which returns *gorm.DB — a concrete struct that
// cannot be intercepted by gomock. Only error paths are unit-tested here.

func TestCreateAlertRoute_RefNotFound_ReturnsError(t *testing.T) {
	// When refUID points to a non-existent node, the service returns an error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockTx := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	route := &model.AlertRoute{Name: "new-sibling", ParentUID: "root-uid"}

	mockDB.EXPECT().Exist(gomock.Any(), "org_id=? and uid=?", int64(12), "root-uid").Return(true, nil)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(db.DB) error) error {
		mockTx.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "ghost-uid", int64(12)).
			Return(fmt.Errorf("record not found"))
		return fn(mockTx)
	})

	_, err := srv.CreateAlertRoute(ctx, &model.CreateAlertRouteRequest{
		AlertRoute: *route,
		RefUID:     "ghost-uid",
		Position:   "before",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ghost-uid")
}

func TestCreateAlertRoute_RefNotSibling_ReturnsError(t *testing.T) {
	// When refUID belongs to a different parent, the service rejects the request.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockTx := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	route := &model.AlertRoute{Name: "new-sibling", ParentUID: "root-uid"}

	mockDB.EXPECT().Exist(gomock.Any(), "org_id=? and uid=?", int64(12), "root-uid").Return(true, nil)

	// ref belongs to a different parent
	refRoute := model.AlertRoute{UID: "ref-uid", ParentUID: "other-parent", SortOrder: 1}
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(db.DB) error) error {
		mockTx.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "ref-uid", int64(12)).
			DoAndReturn(func(dest interface{}, _ ...interface{}) error {
				*(dest.(*model.AlertRoute)) = refRoute
				return nil
			})
		return fn(mockTx)
	})

	_, err := srv.CreateAlertRoute(ctx, &model.CreateAlertRouteRequest{
		AlertRoute: *route,
		RefUID:     "ref-uid",
		Position:   "before",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a sibling")
}

// ─── SortAlertRoutes ─────────────────────────────────────────────────────────

func TestSortAlertRoutes_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockTx := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	routes := []model.AlertRoute{
		{UID: "r1"},
		{UID: "r2"},
	}

	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(db.DB) error) error {
		mockTx.EXPECT().Updates(gomock.Any(), map[string]any{"sort_order": 0, "updated_by": int64(10)},
			"uid=? and org_id=?", "r1", int64(12)).Return(nil)
		mockTx.EXPECT().Updates(gomock.Any(), map[string]any{"sort_order": 1, "updated_by": int64(10)},
			"uid=? and org_id=?", "r2", int64(12)).Return(nil)
		return fn(mockTx)
	})

	err := srv.SortAlertRoutes(ctx, routes)
	assert.NoError(t, err)
}

func TestSortAlertRoutes_DBError_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockTx := db.NewMockDB(ctrl)
	srv := NewAlertRouteService(mockDB)

	routes := []model.AlertRoute{{UID: "r1"}}

	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(db.DB) error) error {
		mockTx.EXPECT().Updates(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("db error"))
		return fn(mockTx)
	})

	err := srv.SortAlertRoutes(ctx, routes)
	assert.Error(t, err)
}
