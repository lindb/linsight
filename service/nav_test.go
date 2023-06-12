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

func TestNavService_GetNavD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	srv := NewNavService(mockDB)
	t.Run("get nav successfully", func(t *testing.T) {
		mockDB.EXPECT().Get(gomock.Any(), "org_id=?", int64(1234)).Return(nil)
		nav, err := srv.GetNavByOrgID(ctx, 1234)
		assert.NotNil(t, nav)
		assert.NoError(t, err)
	})
}

func TestNavService_UpdateNav(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewNavService(mockDB)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get nav failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "org_id=?", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update nav failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "org_id=?", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "org_id=?", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update nav failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "org_id=?", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "org_id=?", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateNav(ctx, 12, &model.Nav{OrgID: 12})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
