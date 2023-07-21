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
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestDashboardService_CreateDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewDashboardService(nil, mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create dashboard failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create dashboard successfully",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.CreateDashboard(ctx, &model.Dashboard{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_DeleteDashboardByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()

	srv := NewDashboardService(nil, mockDB)
	t.Run("delete dashboard failure", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
		err := srv.DeleteDashboardByUID(ctx, "1234")
		assert.Error(t, err)
	})
	t.Run("delete star failure", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
		mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and entity_id=? and entity_type=?", int64(12), "1234", model.DasbboardEntity).
			Return(fmt.Errorf("xx"))
		err := srv.DeleteDashboardByUID(ctx, "1234")
		assert.Error(t, err)
	})
	t.Run("delete successfully", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
		mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and entity_id=? and entity_type=?", int64(12), "1234", model.DasbboardEntity).
			Return(nil)
		err := srv.DeleteDashboardByUID(ctx, "1234")
		assert.NoError(t, err)
	})
}

func TestDashboardService_UpdateDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewDashboardService(nil, mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get dashboard failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update dashboard failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update dashboard failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateDashboard(ctx, &model.Dashboard{UID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_GetDashboardByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	starSrv := NewMockStarService(ctrl)
	srv := NewDashboardService(starSrv, mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get dashboard failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get star failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				starSrv.EXPECT().IsStarred(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get dashboard successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				starSrv.EXPECT().IsStarred(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetDashboardByUID(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_Star(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	starSrv := NewMockStarService(ctrl)
	srv := NewDashboardService(starSrv, mockDB)

	mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
	err := srv.StarDashboard(ctx, "1234")
	assert.Error(t, err)
	mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil).AnyTimes()
	starSrv.EXPECT().Star(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	err = srv.StarDashboard(ctx, "1234")
	assert.Error(t, err)
	starSrv.EXPECT().Unstar(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	err = srv.UnstarDashboard(ctx, "1234")
	assert.NoError(t, err)
}

func TestDashboardService_SerachDashboards(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewDashboardService(nil, mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "count failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "count 0",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			wantErr: false,
		},
		{
			name: "find failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.SearchDashboards(ctx, &model.SearchDashboardRequest{
				Title:     "title",
				Ownership: model.Mine,
				PagingParam: model.PagingParam{
					Limit:  10,
					Offset: 20,
				},
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_GetDashboardsByChartUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	starSrv := NewMockStarService(ctrl)
	srv := NewDashboardService(starSrv, mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get dashboards by chart failure",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get dashboards by chart successfully",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "1234").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetDashboardsByChartUID(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_DeleteProvisioningDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewDashboardService(nil, mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "remove dashboards failure",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"uid in (select dashboard_uid from dashboard_provisionings where org_id=? and name=? and external=?)",
					int64(10), "name", "e").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "remove provisioning dashboards failure",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"uid in (select dashboard_uid from dashboard_provisionings where org_id=? and name=? and external=?)",
					int64(10), "name", "e").Return(nil)
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and name=? and external=?",
					int64(10), "name", "e").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "remove provisioning dashboards successfully",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"uid in (select dashboard_uid from dashboard_provisionings where org_id=? and name=? and external=?)",
					int64(10), "name", "e").Return(nil)
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and name=? and external=?",
					int64(10), "name", "e").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.RemoveProvisioningDashboard(context.TODO(), &model.RemoveProvisioningDashboardRequest{
				Org:      &model.Org{BaseModel: model.BaseModel{ID: 10}},
				Name:     "name",
				External: "e",
			})

			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDashboardService_SaveProvisioningDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewDashboardService(nil, mockDB)

	cases := []struct {
		name    string
		prepare func()
		body    datatypes.JSON
		wantErr bool
	}{
		{
			name: "title empty",
			prepare: func() {
			},
			wantErr: true,
		},
		{
			name: "uid empty",
			body: datatypes.JSON(`{"title":"title"}`),
			prepare: func() {
			},
			wantErr: true,
		},
		{
			name: "check dashboard failure",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update dashboard failure",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(true, nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update provisioning dashboard failure",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(true, nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "dashboard_uid=? and org_id=?", "123", int64(10)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update provisioning dashboard successfully",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(true, nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "dashboard_uid=? and org_id=?", "123", int64(10)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "create dashboard failure",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create provisioning dashboard failure",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create provisioning dashboard successfully",
			body: datatypes.JSON(`{"title":"title","uid":"123"}`),
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=? and org_id=?", "123", int64(10)).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SaveProvisioningDashboard(context.TODO(), &model.SaveProvisioningDashboardRequest{
				Org: &model.Org{BaseModel: model.BaseModel{ID: 10}},
				Dashboard: &model.Dashboard{
					Config: tt.body,
				},
				Provisioning: &model.DashboardProvisioning{
					Name:     "name",
					External: "e",
				},
			})

			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
