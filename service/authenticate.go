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

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

//go:generate mockgen -source=./authenticate.go -destination=./authenticate_mock.go -package=service

// for testing
var (
	randomHexFn = util.RandomHex
)

// AuthenticateService represents user authenticate service interface.
type AuthenticateService interface {
	// Authenticate authenticates the user via username/password.
	Authenticate(ctx context.Context, loginUser *model.LoginUser) (*model.User, error)
	// CreateToken creates a token based on user info.
	CreateToken(ctx context.Context, user *model.User, clientIP, userAgent string) (*model.UserToken, error)
	// LookupToken lookups user token info by given token.
	LookupToken(ctx context.Context, token string) (*model.UserToken, error)
}

// authenticateService implements AuthenticateService interface.
type authenticateService struct {
	userSrv UserService
	db      dbpkg.DB
}

// NewAuthenticateService creates a AuthenticateService instance.
func NewAuthenticateService(userSrv UserService, db dbpkg.DB) AuthenticateService {
	return &authenticateService{
		userSrv: userSrv,
		db:      db,
	}
}

// Authenticate authenticates the user via username/password.
func (srv *authenticateService) Authenticate(ctx context.Context, loginUser *model.LoginUser) (*model.User, error) {
	// TODO: check login too many times
	user, err := srv.userSrv.GetUserByName(ctx, loginUser.Username)
	if err != nil {
		return nil, err
	}
	if user.IsDisabled {
		return nil, constant.ErrUserDisabled
	}
	pwd := util.EncodePassword(loginUser.Password, user.Salt)
	if pwd != user.Password {
		return nil, constant.ErrInvalidCredentials
	}
	return user, nil
}

// CreateToken creates a token based on user info.
func (srv *authenticateService) CreateToken(ctx context.Context, user *model.User, clientIP, userAgent string) (*model.UserToken, error) {
	token, err := randomHexFn(16)
	if err != nil {
		return nil, err
	}
	userToken := &model.UserToken{
		Token:     token,
		UserID:    user.ID,
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}
	err = srv.db.Create(userToken)
	if err != nil {
		return nil, err
	}
	return userToken, nil
}

// LookupToken lookups user token info by given token.
func (srv *authenticateService) LookupToken(ctx context.Context, token string) (*model.UserToken, error) {
	userToken := &model.UserToken{}
	// TODO: add cache????
	if err := srv.db.Get(&userToken, "token=?", token); err != nil {
		return nil, err
	}
	// FIXME: check user token if valid
	return userToken, nil
}
