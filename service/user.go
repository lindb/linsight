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
	"errors"
	"sync"

	"gorm.io/gorm"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

//go:generate mockgen -source=./user.go -destination=./user_mock.go -package=service

// UserService represents user manager interface.
type UserService interface {
	CreateUser(ctx context.Context, user *model.User) (string, error)
	UpdateUser(ctx context.Context, user *model.User) error
	// SavePreference saves the preference of user.
	SavePreference(ctx context.Context, pref *model.Preference) error
	GetSignedUser(ctx context.Context, userID int64) (*model.SignedUser, error)
	GetUserByName(ctx context.Context, nameOrEmail string) (*model.User, error)
	GetUser(ctx context.Context, userID int64) (*model.User, error)
	GetPreference(ctx context.Context, orgID, userID int64) (*model.Preference, error)
}

// userService implements UserService interface.
type userService struct {
	db dbpkg.DB

	cache sync.Map // TODO: use cache
}

func NewUserService(db dbpkg.DB) UserService {
	return &userService{
		db: db,
	}
}

func (srv *userService) GetSignedUser(ctx context.Context, userID int64) (*model.SignedUser, error) {
	val, ok := srv.cache.Load(userID)
	if ok {
		return val.(*model.SignedUser), nil
	}
	user, err := srv.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	orgUser := model.OrgUser{}
	err = srv.db.Get(&orgUser, "org_id=? and user_id=?", user.OrgID, user.ID)
	if err != nil {
		return nil, err
	}
	org := &model.Org{}
	err = srv.db.Get(org, "id=?", user.OrgID)
	if err != nil {
		return nil, err
	}
	pref, err := srv.GetPreference(ctx, user.OrgID, user.ID)
	if err != nil {
		return nil, err
	}

	signedUser := &model.SignedUser{
		User:       user,
		Name:       user.Name,
		Email:      user.Email,
		IsDisabled: user.IsDisabled,
		Org:        org,
		Role:       orgUser.Role,
		Preference: pref,
	}
	srv.cache.Store(userID, signedUser)
	return signedUser, nil
}

func (srv *userService) GetUserByName(ctx context.Context, nameOrEmail string) (*model.User, error) {
	user := &model.User{}
	// TODO: do tolower?
	if err := srv.db.Get(&user, "name=? or email=?", nameOrEmail, nameOrEmail); err != nil {
		return nil, err
	}
	return user, nil
}

func (srv *userService) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	user := &model.User{}
	if err := srv.db.Get(&user, "id=?", userID); err != nil {
		return nil, err
	}
	return user, nil
}

func (srv *userService) UpdateUser(ctx context.Context, user *model.User) error {
	signedUser := util.GetUser(ctx)
	user.UpdatedBy = signedUser.User.ID
	if err := srv.db.Update(&user, "email=?", user.Email); err != nil {
		return err
	}
	return nil
}

func (srv *userService) CreateUser(ctx context.Context, user *model.User) (string, error) {
	uid := uuid.GenerateShortUUID()
	user.UID = uid
	signedUser := util.GetUser(ctx)
	user.CreatedBy = signedUser.User.ID
	user.UpdatedBy = signedUser.User.ID
	if err := srv.db.Update(&user, "email=?", user.Email); err != nil {
		return "", err
	}
	return uid, nil
}

func (srv *userService) GetPreference(ctx context.Context, orgID, userID int64) (*model.Preference, error) {
	pref := model.Preference{}
	if err := srv.db.Get(&pref, "org_id=? and user_id=?", orgID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pref, nil
}

// SavePreference saves the preference of user.
func (srv *userService) SavePreference(ctx context.Context, pref *model.Preference) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	userID := signedUser.User.ID
	prefFormDB, err := srv.GetPreference(ctx, orgID, userID)
	if err != nil {
		return err
	}
	defer func() {
		srv.cache.Delete(userID)
	}()
	if prefFormDB == nil {
		pref.UserID = userID
		pref.OrgID = orgID
		pref.CreatedBy = userID
		pref.UpdatedBy = userID
		return srv.db.Create(pref)
	} else {
		pref.UserID = userID
		pref.OrgID = orgID
		pref.UpdatedBy = userID
		return srv.db.Update(&pref, "org_id=? and user_id=?", orgID, userID)
	}
}
