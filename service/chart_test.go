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

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestChartService_CreateChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewChartService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create chart failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create chart successfully",
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
			_, err := srv.CreateChart(ctx, &model.Chart{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestChartService_SerachCharts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewChartService(mockDB)

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
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), 10, 20).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and title like ? and created_by=?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), 10, 20).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.SearchCharts(ctx, &model.SearchChartRequest{
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

func TestChartService_UpdateChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewChartService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get chart failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update chart failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update chart failure",
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
			err := srv.UpdateChart(ctx, &model.Chart{UID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestChartService_DeleteChartByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()

	srv := NewChartService(mockDB)
	t.Run("delete successfully", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
		err := srv.DeleteChartByUID(ctx, "1234")
		assert.NoError(t, err)
	})
}

func TestChartService_GetChartByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewChartService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get chart failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get chart successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetChartByUID(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestChartService_LinkChartsToDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewChartService(mockDB)
	cases := []struct {
		name      string
		dashboard *model.Dashboard
		prepare   func()
		wantErr   bool
	}{
		{
			name: "get chart ids from dashboard failure",
			dashboard: &model.Dashboard{
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{}}]}`),
			},
			prepare: func() {},
			wantErr: true,
		},
		{
			name: "delete old links failure",
			dashboard: &model.Dashboard{
				UID:    "123",
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{"uid":"abc"}}]}`),
			},
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and kind=? and target_uid=?",
					int64(12), model.DashboardLink, "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create links failure",
			dashboard: &model.Dashboard{
				UID:    "123",
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{"uid":"abc"}}]}`),
			},
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and kind=? and target_uid=?",
					int64(12), model.DashboardLink, "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create links successfully",
			dashboard: &model.Dashboard{
				UID:    "123",
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{"uid":"abc"}}]}`),
			},
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and kind=? and target_uid=?",
					int64(12), model.DashboardLink, "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.LinkChartsToDashboard(ctx, tt.dashboard)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestChartService_UnlinkChartFromDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewChartService(mockDB)
	mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and kind=? and source_uid=? and target_uid=?",
		int64(12), model.DashboardLink, "123", "abc",
	).Return(nil)
	err := srv.UnlinkChartFromDashboard(ctx, "123", "abc")
	assert.NoError(t, err)
}
