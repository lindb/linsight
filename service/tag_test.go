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

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestTagService_FindTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewTagService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "find tags failure",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term like ?", int64(12), "name%").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find tags successfully",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term like ?", int64(12), "name%").
					DoAndReturn(func(rs *[]model.Tag, _ string, _ int64, _ string) error {
						*rs = append(*rs, model.Tag{Term: "tag"})
						return nil
					})
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.FindTags(ctx, "name")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTagService_SaveTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewTagService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "find tags failure",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term in ?", int64(12), []string{"tags", "name"}).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create tags failure",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term in ?", int64(12), []string{"tags", "name"}).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "delete old tag relations failure",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term in ?", int64(12), []string{"tags", "name"}).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil).MaxTimes(2)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and type=? and resource_uid=?",
					int64(12), model.DashboardResource, "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create new tag relations failure",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term in ?", int64(12), []string{"tags", "name"}).
					DoAndReturn(func(rs *[]model.Tag, _ string, _ int64, _ []string) error {
						*rs = append(*rs, model.Tag{Term: "tags"})
						return nil
					})
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and type=? and resource_uid=?",
					int64(12), model.DashboardResource, "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "save tags successfully",
			prepare: func() {
				mockDB.EXPECT().Find(gomock.Any(), "org_id=? and term in ?", int64(12), []string{"tags", "name"}).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil).MaxTimes(2)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=? and type=? and resource_uid=?",
					int64(12), model.DashboardResource, "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil).MaxTimes(2)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SaveTags(12, []string{"tags", "name"}, "123", model.DashboardResource)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
