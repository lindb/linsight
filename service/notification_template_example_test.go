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
	"github.com/stretchr/testify/require"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestNotificationTemplateExampleService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNotificationTemplateExampleService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "create successfully",
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
			uid, err := srv.CreateNotificationTemplateExample(ctx, &model.NotificationTemplateExample{
				Name:    "CPU firing example",
				Payload: `{"status":"firing","alerts":[{"ruleTitle":"HighCPU"}]}`,
			})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, uid)
			}
		})
	}
}

func TestNotificationTemplateExampleService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNotificationTemplateExampleService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get example failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(fmt.Errorf("not found"))
			},
			wantErr: true,
		},
		{
			name: "update failure",
			prepare: func() {
				// DoAndReturn populates the out pointer so fromDB is non-nil after Get.
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).
					DoAndReturn(func(out any, where ...any) error {
						v := out.(**model.NotificationTemplateExample)
						*v = &model.NotificationTemplateExample{UID: "uid-1", Name: "Old"}
						return nil
					})
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "update successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).
					DoAndReturn(func(out any, where ...any) error {
						v := out.(**model.NotificationTemplateExample)
						*v = &model.NotificationTemplateExample{UID: "uid-1", Name: "Old"}
						return nil
					})
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateNotificationTemplateExample(ctx, &model.NotificationTemplateExample{
				UID:  "uid-1",
				Name: "Updated name",
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestNotificationTemplateExampleService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNotificationTemplateExampleService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "delete failure",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "delete successfully",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.DeleteNotificationTemplateExampleByUID(ctx, "uid-1")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestNotificationTemplateExampleService_GetByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNotificationTemplateExampleService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(fmt.Errorf("not found"))
			},
			wantErr: true,
		},
		{
			name: "get successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "uid-1", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetNotificationTemplateExampleByUID(ctx, "uid-1")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestNotificationTemplateExampleService_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNotificationTemplateExampleService(mockDB)

	cases := []struct {
		name      string
		req       *model.SearchNotificationTemplateExampleRequest
		prepare   func()
		wantTotal int64
		wantErr   bool
	}{
		{
			name: "count failure",
			req:  &model.SearchNotificationTemplateExampleRequest{},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=?", int64(12)).Return(int64(0), fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "empty result",
			req:  &model.SearchNotificationTemplateExampleRequest{},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=?", int64(12)).Return(int64(0), nil)
			},
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name: "find failure",
			req:  &model.SearchNotificationTemplateExampleRequest{},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=?", int64(12)).Return(int64(2), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 0, 20, "id desc", "org_id=?", int64(12)).Return(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "search successfully without filter",
			req:  &model.SearchNotificationTemplateExampleRequest{},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=?", int64(12)).Return(int64(2), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 0, 20, "id desc", "org_id=?", int64(12)).Return(nil)
			},
			wantTotal: 2,
			wantErr:   false,
		},
		{
			name: "search with name filter",
			req:  &model.SearchNotificationTemplateExampleRequest{Name: "CPU"},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and name like ?", int64(12), "CPU%").Return(int64(1), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 0, 20, "id desc", "org_id=? and name like ?", int64(12), "CPU%").Return(nil)
			},
			wantTotal: 1,
			wantErr:   false,
		},
		{
			name: "search with custom pagination",
			req: &model.SearchNotificationTemplateExampleRequest{
				PagingParam: model.PagingParam{Offset: 10, Limit: 5},
			},
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=?", int64(12)).Return(int64(15), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 10, 5, "id desc", "org_id=?", int64(12)).Return(nil)
			},
			wantTotal: 15,
			wantErr:   false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, total, err := srv.SearchNotificationTemplateExamples(ctx, tt.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTotal, total)
			}
		})
	}
}
