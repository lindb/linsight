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
	"github.com/lindb/common/pkg/encoding"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestIntegrationService_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	srv := NewIntegrationService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "unmarshal cmp failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "check exist failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", gomock.Any()).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", gomock.Any()).Return(true, nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "initialize successfully",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", gomock.Any()).Return(false, nil).AnyTimes()
				mockDB.EXPECT().Create(gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
			}()
			tt.prepare()
			err := srv.Initialize()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestIntegrationService_GetIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewIntegrationService(mockDB)

	t.Run("load failure", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(fmt.Errorf("err"))
		tree, err := srv.GetIntegrations(context.TODO())
		assert.Error(t, err)
		assert.Nil(t, tree)
	})

	t.Run("load successfully", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(nil)
		tree, err := srv.GetIntegrations(context.TODO())
		assert.NoError(t, err)
		assert.Nil(t, tree)
	})
}

func TestIntegrationService_DisconnectSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Delete(gomock.Any(),
		"org_id=? and source_id=? and type=?",
		int64(12), "123", model.DashboardConnection).Return(nil)
	srv := NewIntegrationService(mockDB)
	assert.Nil(t, srv.DisconnectSource(ctx, "123", model.DashboardConnection))
}

func TestIntegrationService_ConnectSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewIntegrationService(mockDB)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "delete old failure",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and source_id=? and type=?",
					int64(12), "abc", model.ChartConnection).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "connect successfully",
			prepare: func() {
				mockDB.EXPECT().Delete(gomock.Any(),
					"org_id=? and source_id=? and type=?",
					int64(12), "abc", model.ChartConnection).Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.ConnectSource(ctx, "123", "abc", model.ChartConnection)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
