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

func TestChartService_getChartByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srvINF := NewChartService(mockDB)
	srv := srvINF.(*chartService)
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
			_, err := srv.getChartByUID(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
