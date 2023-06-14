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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

func TestAuthenticateService_Authenticate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := NewMockUserService(ctrl)
	mockDB := db.NewMockDB(ctrl)
	authSrv := NewAuthenticateService(userSrv, mockDB)

	cases := []struct {
		name    string
		prepare func()
		assert  func(user *model.User, err error)
	}{
		{
			name: "get user failure",
			prepare: func() {
				userSrv.EXPECT().GetUserByName(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(user *model.User, err error) {
				assert.Nil(t, user)
				assert.Error(t, err)
			},
		},
		{
			name: "user is disabled",
			prepare: func() {
				disabled := true
				userSrv.EXPECT().GetUserByName(gomock.Any(), gomock.Any()).Return(&model.User{IsDisabled: &disabled}, nil)
			},
			assert: func(user *model.User, err error) {
				assert.Nil(t, user)
				assert.Equal(t, constant.ErrUserDisabled, err)
			},
		},
		{
			name: "invalid credentials",
			prepare: func() {
				disabled := false
				userSrv.EXPECT().GetUserByName(gomock.Any(), gomock.Any()).Return(&model.User{Password: "pwd", IsDisabled: &disabled}, nil)
			},
			assert: func(user *model.User, err error) {
				assert.Nil(t, user)
				assert.Equal(t, constant.ErrInvalidCredentials, err)
			},
		},
		{
			name: "successfully",
			prepare: func() {
				disabled := false
				userSrv.EXPECT().GetUserByName(gomock.Any(), gomock.Any()).
					Return(&model.User{Password: util.EncodePassword("pwd", "123456"), Salt: "123456", IsDisabled: &disabled}, nil)
			},
			assert: func(user *model.User, err error) {
				assert.NotNil(t, user)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			tt.prepare()
			user, err := authSrv.Authenticate(context.TODO(), &model.LoginUser{
				Password: "pwd",
			})
			tt.assert(user, err)
		})
	}
}

func TestAuthenticateService_CreateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	authSrv := NewAuthenticateService(nil, mockDB)

	cases := []struct {
		name    string
		prepare func()
		assert  func(user *model.UserToken, err error)
	}{
		{
			name: "gen token failure",
			prepare: func() {
				randomHexFn = func(n int) (string, error) {
					return "", fmt.Errorf("err")
				}
			},
			assert: func(user *model.UserToken, err error) {
				assert.Nil(t, user)
				assert.Error(t, err)
			},
		},
		{
			name: "create token failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(user *model.UserToken, err error) {
				assert.Nil(t, user)
				assert.Error(t, err)
			},
		},
		{
			name: "successfully",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			assert: func(user *model.UserToken, err error) {
				assert.NotNil(t, user)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			defer func() {
				randomHexFn = util.RandomHex
			}()
			tt.prepare()
			user, err := authSrv.CreateToken(context.TODO(), &model.User{}, "1.1.1.1", "ie")
			tt.assert(user, err)
		})
	}
}
