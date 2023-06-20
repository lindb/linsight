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
	"strings"
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
	// CreateUser creates a new user, returns the user's uid, if fail returns error.
	CreateUser(ctx context.Context, user *model.CreateUserRequest) (string, error)
	// UpdateUser updates user basic information.
	UpdateUser(ctx context.Context, user *model.User) error
	// GetUserByUID returns user basic information by given uid.
	GetUserByUID(ctx context.Context, uid string) (*model.User, error)
	// SearchUser returns user list by query condition.
	SearchUser(ctx context.Context, req *model.SearchUserRequest) (users []model.User, total int64, err error)
	// DisableUser disables user.
	DisableUser(ctx context.Context, uid string) error
	// EnableUser enbles user.
	EnableUser(ctx context.Context, uid string) error
	// GetSignedUser returns user basic information by user id.
	GetSignedUser(ctx context.Context, userID int64) (*model.SignedUser, error)
	// GetUserByName returns user by given username or email.
	GetUserByName(ctx context.Context, nameOrEmail string) (*model.User, error)

	// GetPreference returns the preference of current signed user for current org.
	GetPreference(ctx context.Context) (*model.Preference, error)
	// GetPreference returns the preference of current signed user for current org.
	SavePreference(ctx context.Context, pref *model.Preference) error

	// ChangePassword changes user password.
	ChangePassword(ctx context.Context, changePWD *model.ChangeUserPassword) error
	// ResetPassword resets user password.
	ResetPassword(ctx context.Context, resetPWD *model.ResetUserPassword) error

	// GetOrgListByUserUID returns org list which user belongs.
	GetOrgListByUserUID(ctx context.Context, uid string) ([]model.UserOrgInfo, error)
	// AddOrg creates user's org(user join org.).
	AddOrg(ctx context.Context, userOrg *model.UserOrgInfo) error
	// RemoveOrg removes user's org(user leave org.).
	RemoveOrg(ctx context.Context, userOrg *model.UserOrgInfo) error
	// UpdateOrg updates user's org(such as role etc.)
	UpdateOrg(ctx context.Context, userOrg *model.UserOrgInfo) error
	// SwitchOrg switches target org.
	SwitchOrg(ctx context.Context, orgUID string) error
}

// userService implements the UserService interface.
type userService struct {
	db     dbpkg.DB
	orgSrv OrgService

	cache sync.Map // TODO: use cache
}

// NewUserService creates an UserService instance.
func NewUserService(db dbpkg.DB, orgSrv OrgService) UserService {
	return &userService{
		db:     db,
		orgSrv: orgSrv,
		cache:  sync.Map{},
	}
}

// GetSignedUser returns user basic information by user id.
func (srv *userService) GetSignedUser(ctx context.Context, userID int64) (*model.SignedUser, error) {
	user, err := srv.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	signedUser := &model.SignedUser{
		User:       user,
		UID:        user.UID,
		Name:       user.Name,
		UserName:   user.UserName,
		Email:      user.Email,
		IsDisabled: user.IsDisabled,
	}
	if user.OrgID > 0 {
		// if user belongs org
		var org model.Org
		var orgUser model.OrgUser
		err = srv.db.Get(&orgUser, "org_id=? and user_id=?", user.OrgID, user.ID)
		if err != nil {
			return nil, err
		}
		err = srv.db.Get(&org, "id=?", user.OrgID)
		if err != nil {
			return nil, err
		}
		signedUser.Org = &org
		// FIXME: need add Lin Role check
		signedUser.Role = orgUser.Role
	}
	pref, err := srv.getPreference(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if pref == nil {
		// if preference not found, set default value
		pref = &model.DefaultUserPreference
	}
	signedUser.Preference = pref

	srv.cache.Store(userID, signedUser)
	return signedUser, nil
}

// GetUserByName returns user by given username or email.
func (srv *userService) GetUserByName(ctx context.Context, nameOrEmail string) (*model.User, error) {
	user := &model.User{}
	if err := srv.db.Get(&user, "user_name=? or email=?", nameOrEmail, nameOrEmail); err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser creates a new user, returns the user's uid, if fail returns error.
func (srv *userService) CreateUser(ctx context.Context, user *model.CreateUserRequest) (string, error) {
	uid := uuid.GenerateShortUUID()
	// encode password
	salt, _ := util.GetRandomString(10)
	pwd := util.EncodePassword(user.Password, salt)
	if user.Name == "" {
		user.Name = user.UserName
	}
	newUser := &model.User{
		UID:      uid,
		Password: pwd,
		Salt:     salt,
		Name:     user.Name,
		UserName: user.UserName,
		Email:    user.Email,
	}
	signedUser := util.GetUser(ctx)
	newUser.CreatedBy = signedUser.User.ID
	newUser.UpdatedBy = signedUser.User.ID
	if err := srv.db.Create(newUser); err != nil {
		return "", err
	}
	return uid, nil
}

// UpdateUser updates user basic information.
func (srv *userService) UpdateUser(ctx context.Context, user *model.User) error {
	userFromDB, err := srv.GetUserByUID(ctx, user.UID)
	if err != nil {
		return err
	}
	signedUser := util.GetUser(ctx)
	// FIXME: check UserName if exist
	userFromDB.UserName = user.UserName
	userFromDB.Name = user.Name
	userFromDB.Email = user.Email
	userFromDB.UpdatedBy = signedUser.User.ID
	if err := srv.db.Update(userFromDB, "uid=?", user.UID); err != nil {
		return err
	}
	return nil
}

// GetUserByUID returns user basic information by given uid.
func (srv *userService) GetUserByUID(ctx context.Context, uid string) (*model.User, error) {
	user := &model.User{}
	if err := srv.db.Get(user, "uid=?", uid); err != nil {
		return nil, err
	}
	return user, nil
}

// SearchUser returns user list by query condition.
func (srv *userService) SearchUser(ctx context.Context, req *model.SearchUserRequest) (users []model.User, total int64, err error) {
	conditions := []string{}
	params := []any{}
	if req.Query != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Query+"%")
		conditions = append(conditions, "user_name like ?")
		params = append(params, req.Query+"%")
		conditions = append(conditions, "email like ?")
		params = append(params, req.Query+"%")
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	where := strings.Join(conditions, " or ")
	count, err := srv.db.Count(&model.User{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	if err := srv.db.FindForPaging(&users, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

// DisableUser disables user.
func (srv *userService) DisableUser(ctx context.Context, uid string) error {
	return srv.setUserDisableState(ctx, uid, true)
}

// EnableUser enbles user.
func (srv *userService) EnableUser(ctx context.Context, uid string) error {
	return srv.setUserDisableState(ctx, uid, false)
}

// GetPreference returns the preference of current signed user for current org.
func (srv *userService) GetPreference(ctx context.Context) (*model.Preference, error) {
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	return srv.getPreference(ctx, userID)
}

// SavePreference saves the preference of signed user for current org.
func (srv *userService) SavePreference(ctx context.Context, pref *model.Preference) error {
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	prefFormDB, err := srv.getPreference(ctx, userID)
	if err != nil {
		return err
	}
	defer func() {
		srv.cache.Delete(userID)
	}()
	if prefFormDB == nil {
		pref.UserID = userID
		pref.CreatedBy = userID
		pref.UpdatedBy = userID
		return srv.db.Create(pref)
	} else {
		return srv.db.Updates(&model.Preference{}, map[string]any{
			"theme":      pref.Theme,
			"collapsed":  pref.Collapsed,
			"home_page":  pref.HomePage,
			"updated_by": userID,
		}, "user_id=?", userID)
	}
}

// ChangePassword changes user password.
func (srv *userService) ChangePassword(ctx context.Context, changePWD *model.ChangeUserPassword) error {
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	user, err := srv.getUser(ctx, userID)
	if err != nil {
		return err
	}
	pwd := util.EncodePassword(changePWD.OldPassword, user.Salt)
	// check old password
	if pwd != user.Password {
		return errors.New("old password invalid")
	}
	newPWD := util.EncodePassword(changePWD.NewPassword, user.Salt)
	return srv.db.UpdateSingle(&model.User{}, "password", newPWD, "id=?", userID)
}

// ResetPassword resets user password.
func (srv *userService) ResetPassword(ctx context.Context, resetPWD *model.ResetUserPassword) error {
	user, err := srv.GetUserByUID(ctx, resetPWD.UserUID)
	if err != nil {
		return err
	}
	newPWD := util.EncodePassword(resetPWD.Password, user.Salt)
	return srv.db.UpdateSingle(&model.User{}, "password", newPWD, "uid=?", resetPWD.UserUID)
}

// GetOrgListByUserUID returns org list which user belongs.
func (srv *userService) GetOrgListByUserUID(ctx context.Context, uid string) (rs []model.UserOrgInfo, err error) {
	user, err := srv.GetUserByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	sql := `
	select 
		u.uid as user_uid,o.uid as org_uid,o.name as org_name,ou.role 
	from users u,orgs o,org_users ou 
		where u.id=ou.user_id and o.id=ou.org_id and u.id=?`
	err = srv.db.ExecRaw(&rs, sql, user.ID)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

// AddOrg creates user's org(user join org.).
func (srv *userService) AddOrg(ctx context.Context, userOrg *model.UserOrgInfo) error {
	user, org, err := srv.getUserAndOrg(ctx, userOrg.UserUID, userOrg.OrgUID)
	if err != nil {
		return err
	}
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	uo := &model.OrgUser{
		UserID: user.ID,
		OrgID:  org.ID,
		Role:   userOrg.Role,
	}
	uo.CreatedBy = userID
	uo.UpdatedBy = userID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		if err := tx.Create(uo); err != nil {
			return err
		}
		if user.OrgID <= 0 {
			// set default select org.
			user.OrgID = org.ID
			return tx.Update(user, "id=?", user.ID)
		}
		return nil
	})
}

// RemoveOrg removes user's org(user leave org.).
func (srv *userService) RemoveOrg(ctx context.Context, userOrg *model.UserOrgInfo) error {
	user, org, err := srv.getUserAndOrg(ctx, userOrg.UserUID, userOrg.OrgUID)
	if err != nil {
		return err
	}
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		if err := tx.Delete(&model.OrgUser{}, "org_id=? and user_id=?", org.ID, user.ID); err != nil {
			return err
		}
		orgUsers := []model.OrgUser{}
		err := tx.Find(&orgUsers, "user_id=?", user.ID)
		if err != nil {
			return err
		}
		orgID := int64(0)
		if len(orgUsers) > 0 {
			// set default select org.
			orgID = orgUsers[0].OrgID
		}
		return tx.UpdateSingle(&model.User{}, "org_id", orgID, "id=?", user.ID)
	})
}

// UpdateOrg updates user's org(such as role etc.)
func (srv *userService) UpdateOrg(ctx context.Context, userOrg *model.UserOrgInfo) error {
	user, org, err := srv.getUserAndOrg(ctx, userOrg.UserUID, userOrg.OrgUID)
	if err != nil {
		return err
	}
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	uo := &model.OrgUser{
		Role: userOrg.Role,
	}
	uo.UpdatedBy = userID
	return srv.db.Update(uo, "org_id=? and user_id=?", org.ID, user.ID)
}

// SwitchOrg switches target org.
func (srv *userService) SwitchOrg(ctx context.Context, orgUID string) error {
	signedUser := util.GetUser(ctx)
	var org model.Org
	err := srv.db.Get(&org, "uid=?", orgUID)
	if err != nil {
		return err
	}
	// check target org if exist
	exist, err := srv.db.Exist(&model.OrgUser{}, "user_id=? and org_id=?", signedUser.User.ID, org.ID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("organization not exist")
	}
	return srv.db.UpdateSingle(&model.User{}, "org_id", org.ID, "id=?", signedUser.User.ID)
}

// getPreference returns the preference by given org/user.
func (srv *userService) getPreference(_ context.Context, userID int64) (*model.Preference, error) {
	pref := model.Preference{}
	if err := srv.db.Get(&pref, "user_id=?", userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pref, nil
}

// setUserDisableState sets user's disable state.
func (srv *userService) setUserDisableState(_ context.Context, uid string, disable bool) error {
	return srv.db.UpdateSingle(&model.User{}, "is_disabled", disable, "uid=?", uid)
}

// getUserAndOrg returns user and org by given user/org uid.
func (srv *userService) getUserAndOrg(ctx context.Context, userUID, orgUID string) (*model.User, *model.Org, error) {
	user, err := srv.GetUserByUID(ctx, userUID)
	if err != nil {
		return nil, nil, err
	}
	org, err := srv.orgSrv.GetOrgByUID(ctx, orgUID)
	if err != nil {
		return nil, nil, err
	}
	return user, org, nil
}

// getUser returns user by id.
func (srv *userService) getUser(_ context.Context, userID int64) (*model.User, error) {
	user := &model.User{}
	if err := srv.db.Get(user, "id=?", userID); err != nil {
		return nil, err
	}
	return user, nil
}
