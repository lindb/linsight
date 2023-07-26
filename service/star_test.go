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

func TestStarService_Star(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewStarService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "check if star failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "user_id=? and org_id=? and resource_uid=? and resource_type=?",
					int64(10), int64(12), "100", model.DashboardResource).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "already starred",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "user_id=? and org_id=? and resource_uid=? and resource_type=?",
					int64(10), int64(12), "100", model.DashboardResource).Return(true, nil)
			},
			wantErr: false,
		},
		{
			name: "star failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "user_id=? and org_id=? and resource_uid=? and resource_type=?",
					int64(10), int64(12), "100", model.DashboardResource).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.Star(ctx, "100", model.DashboardResource)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestStarService_Unstar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewStarService(mockDB)
	mockDB.EXPECT().Delete(gomock.Any(), "user_id=? and org_id=? and resource_uid=? and resource_type=?",
		int64(10), int64(12), "100", model.DashboardResource).Return(nil)
	err := srv.Unstar(ctx, "100", model.DashboardResource)
	assert.NoError(t, err)
}

func TestStarService_IsStarred(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewStarService(mockDB)
	mockDB.EXPECT().Exist(gomock.Any(), "user_id=? and org_id=? and resource_uid=? and resource_type=?",
		int64(10), int64(12), "100", model.DashboardResource).Return(true, nil)
	starrted, err := srv.IsStarred(ctx, "100", model.DashboardResource)
	assert.NoError(t, err)
	assert.True(t, starrted)
}
