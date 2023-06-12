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

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestOrgService_SearchOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewOrgService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "count failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ?",
					gomock.Any()).Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "count 0",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ?",
					gomock.Any()).Return(int64(0), nil)
			},
			wantErr: false,
		},
		{
			name: "find failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ?",
					gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "name like ?",
					gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ?",
					gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "name like ?",
					gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.SearchOrg(ctx, &model.SearchOrgRequest{
				Name: "name",
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

func TestOrgService_CreateOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewOrgService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create org failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create org successfully",
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
			_, err := srv.CreateOrg(ctx, &model.Org{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestOrgService_UpdateOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewOrgService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=?", "1234").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateOrg(ctx, &model.Org{UID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
func TestOrgService_DeleteOrgByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewOrgService(mockDB)
	t.Run("delete successfully", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=?", "1234").Return(nil)
		err := srv.DeleteOrgByUID(ctx, "1234")
		assert.NoError(t, err)
	})
}

func TestOrgService_GetOrgByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	srv := NewOrgService(mockDB)
	t.Run("get org successfully", func(t *testing.T) {
		mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
		org, err := srv.GetOrgByUID(ctx, "1234")
		assert.NotNil(t, org)
		assert.NoError(t, err)
	})
}
