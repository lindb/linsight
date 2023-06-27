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

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func TestBootAPI_Boot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	cmpSrv := service.NewMockComponentService(ctrl)
	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()

	api := NewBootAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
		CmpSrv:        cmpSrv,
		UserSrv:       userSrv,
	})
	r.GET("/boot", api.Boot)
	signedUser := &model.SignedUser{
		Org: &model.Org{
			BaseModel: model.BaseModel{
				ID: 12,
			},
		},
		User: &model.User{
			BaseModel: model.BaseModel{
				ID: 10,
			},
		},
		Preference: &model.DefaultUserPreference,
	}

	cases := []struct {
		name    string
		ctx     context.Context
		prepare func()
	}{
		{
			name: "get signed user fail",
			ctx:  context.WithValue(context.TODO(), constant.LinSightSignedKey, signedUser),
			prepare: func() {
				userSrv.EXPECT().GetSignedUser(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name: "user belong one org but get resource fail",
			ctx:  context.WithValue(context.TODO(), constant.LinSightSignedKey, signedUser),
			prepare: func() {
				userSrv.EXPECT().GetSignedUser(gomock.Any(), gomock.Any()).Return(signedUser, nil)
				datasourceSrv.EXPECT().GetDatasources(gomock.Any()).Return(nil, fmt.Errorf("err"))
				cmpSrv.EXPECT().GetComponentTreeByCurrentOrg(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name: "user belong one org successfully",
			ctx: context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
				Org: &model.Org{
					BaseModel: model.BaseModel{
						ID: 12,
					},
				},
				User: &model.User{
					BaseModel: model.BaseModel{
						ID: 10,
					},
				},
				Preference: &model.DefaultUserPreference,
			}),
			prepare: func() {
				userSrv.EXPECT().GetSignedUser(gomock.Any(), gomock.Any()).Return(signedUser, nil)
				datasourceSrv.EXPECT().GetDatasources(gomock.Any()).Return(nil, nil)
				cmpSrv.EXPECT().GetComponentTreeByCurrentOrg(gomock.Any()).Return(model.Components{}, nil)
			},
		},
		{
			name: "user no org",
			ctx: context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
				Preference: &model.DefaultUserPreference,
				User: &model.User{
					BaseModel: model.BaseModel{
						ID: 10,
					},
				},
			}),
			prepare: func() {
				signedUser.Org = nil
				userSrv.EXPECT().GetSignedUser(gomock.Any(), gomock.Any()).Return(signedUser, nil)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(tt.ctx, http.MethodGet, "/boot", http.NoBody)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			if tt.prepare != nil {
				tt.prepare()
			}
			r.ServeHTTP(resp, req)
		})
	}
}
