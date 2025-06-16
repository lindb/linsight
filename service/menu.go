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

	"github.com/lindb/common/pkg/encoding"
	"gorm.io/gorm"

	"github.com/lindb/linsight"
	"github.com/lindb/linsight/accesscontrol"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

//go:generate mockgen -source=./menu.go -destination=./menu_mock.go -package=service

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
)

// MenuService represents platform menu manager interface.
type MenuService interface {
	// Initialize initializes supported platform menus.
	Initialize() error
	// LoadMenuTree returns supported platform menus.
	LoadMenuTree(ctx context.Context) (model.Menus, error)
	// CreateMenu creates a new menu.
	CreateMenu(ctx context.Context, menu *model.Menu) (string, error)
	// UpdateMenu updates menu.
	UpdateMenu(ctx context.Context, menu *model.Menu) error
	// DeleteMenuByUID deletes menu than manual created.
	DeleteMenuByUID(ctx context.Context, uid string) error

	// SortMenus sorts menu list.
	SortMenus(ctx context.Context, menus model.Menus) error

	// GetMenuTreeByCurrentOrg returns menu tree that current user can access of current org.
	GetMenuTreeByCurrentOrg(ctx context.Context) (model.Menus, error)
	// GetOrgMenus returns menu list for org.
	GetOrgMenus(ctx context.Context, orgUID string) ([]model.OrgMenuInfo, error)
	// SaveOrgMenus creates or removes menus for menus of org.
	SaveOrgMenus(ctx context.Context, orgUID string, menus []model.OrgMenuInfo) error
	// UpdateRolesOfOrgMenus updates roles for current org.
	UpdateRolesOfOrgMenus(ctx context.Context, menus []model.OrgMenuInfo) error
}

// menuService implements MenuService interface.
type menService struct {
	db dbpkg.DB

	orgSrv       OrgService
	authorizeSrv AuthorizeService
}

// NewMenuService creates a MenuService instance.
func NewMenuService(db dbpkg.DB, orgSrv OrgService, authorizeSrv AuthorizeService) MenuService {
	return &menService{
		db:           db,
		orgSrv:       orgSrv,
		authorizeSrv: authorizeSrv,
	}
}

// Initialize initializes supported platform menus.
func (srv *menService) Initialize() error {
	mainOrg, err := srv.orgSrv.GetOrgByName(context.TODO(), constant.AdminOrgName)
	if err != nil {
		return err
	}
	var menus model.Menus
	if err := jsonUnmarshalFn([]byte(linsight.Menus), &menus); err != nil {
		return err
	}
	return srv.saveMenuTree(menus, mainOrg.ID)
}

// LoadMenuTree returns supported platform menus.
func (srv *menService) LoadMenuTree(ctx context.Context) (model.Menus, error) {
	var menus model.Menus
	if err := srv.db.Find(&menus); err != nil {
		return nil, err
	}
	return menus.ToTree(), nil
}

// UpdateMenu updates menu.
func (srv *menService) UpdateMenu(ctx context.Context, menu *model.Menu) error {
	user := util.GetUser(ctx)
	return srv.db.Updates(&model.Menu{}, map[string]any{
		"label":       menu.Label,
		"path":        menu.Path,
		"icon":        menu.Icon,
		"role":        menu.Role,
		"page":        menu.Page,
		"order":       menu.Order,
		"page_config": menu.PageConfig,
		"parent_uid":  menu.ParentUID,
		"updated_by":  user.User.ID,
	}, "uid=?", menu.UID)
}

// CreateMenu creates a new menu.
func (srv *menService) CreateMenu(ctx context.Context, menu *model.Menu) (string, error) {
	if menu.ParentUID != "" {
		// check parent if exist
		exist, err := srv.db.Exist(&model.Menu{}, "uid=?", menu.ParentUID)
		if err != nil {
			return "", err
		}
		if !exist {
			return "", errors.New("parent menu not exist")
		}
	}
	user := util.GetUser(ctx)
	menu.UID = uuid.GenerateShortUUID()
	menu.UpdatedBy = user.User.ID
	menu.CreatedBy = user.User.ID
	if err := srv.db.Create(&menu); err != nil {
		return "", err
	}
	return menu.UID, nil
}

// DeleteMenuByUID deletes menu than manual created.
func (srv *menService) DeleteMenuByUID(ctx context.Context, uid string) error {
	// check menu if has children
	count, err := srv.db.Count(&model.Menu{}, "parent_uid=?", uid)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("menu has children")
	}
	return srv.db.Delete(&model.Menu{}, "uid=?", uid)
}

// SortMenus sorts menu list.
func (srv *menService) SortMenus(ctx context.Context, menus model.Menus) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for i, menu := range menus {
			if err := tx.Updates(model.Menu{}, map[string]any{
				"order":      i,
				"parent_uid": menu.ParentUID,
				"updated_by": user.User.ID,
			}, "uid=?", menu.UID); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMenuTreeByCurrentOrg returns menu tree that current user can access of current org.
func (srv *menService) GetMenuTreeByCurrentOrg(ctx context.Context) (model.Menus, error) {
	var menus model.Menus
	user := util.GetUser(ctx)
	sql := `select  
		c.uid,c.label,c.path,c.icon,c.page,c.page_config,c.parent_uid,oc.role,c.'order' 
	from menus c, org_menus oc where c.id=oc.menu_id and oc.org_id=?
	`
	if err := srv.db.ExecRaw(&menus, sql, user.Org.ID); err != nil {
		return nil, err
	}
	var aclParamList []model.ResourceACLParam
	for _, navItem := range menus {
		aclParamList = append(aclParamList, model.ResourceACLParam{
			Role:     user.Role,
			OrgID:    user.Org.ID,
			Category: accesscontrol.Menu,
			Resource: navItem.UID,
			Action:   accesscontrol.Write,
		})
	}
	// check acl
	result, err := srv.authorizeSrv.CheckResourcesACL(aclParamList)
	if err != nil {
		return nil, err
	}
	var accessPages model.Menus
	for i, ok := range result {
		if ok {
			accessPages = append(accessPages, menus[i])
		}
	}
	return accessPages.ToTree(), nil
}

// GetOrgMenus returns menu list for org.
func (srv *menService) GetOrgMenus(ctx context.Context, orgUID string) ([]model.OrgMenuInfo, error) {
	var menus []model.OrgMenuInfo
	sql := `select 
		c.uid as menu_uid,oc.role 
	from menus c, org_menus oc,orgs o 
	where c.page <> '' and c.id=oc.menu_id and oc.org_id=o.id and o.uid=?
	`
	if err := srv.db.ExecRaw(&menus, sql, orgUID); err != nil {
		return nil, err
	}
	return menus, nil
}

// SaveOrgMenus creates or removes menus for menus of org.
func (srv *menService) SaveOrgMenus(ctx context.Context, orgUID string, menus []model.OrgMenuInfo) error {
	user := util.GetUser(ctx)
	org, err := srv.orgSrv.GetOrgByUID(ctx, orgUID)
	if err != nil {
		return err
	}
	var acl []model.ResourceACLParam
	if err := srv.db.Transaction(func(tx dbpkg.DB) error {
		// remove menus
		if err := tx.Delete(&model.OrgMenu{}, "org_id=?", org.ID); err != nil {
			return err
		}
		// create menus
		for _, menu := range menus {
			menuFromDB := &model.Menu{}
			if err := tx.Get(menuFromDB, "uid=?", menu.MenuUID); err != nil {
				return err
			}
			if err := tx.Create(&model.OrgMenu{
				OrgID:  org.ID,
				MenuID: menuFromDB.ID,
				Role:   menu.Role,
				BaseModel: model.BaseModel{
					CreatedBy: user.User.ID,
					UpdatedBy: user.User.ID,
				},
			}); err != nil {
				return err
			}
			acl = append(acl, model.ResourceACLParam{
				Role:     menuFromDB.Role,
				OrgID:    org.ID,
				Category: accesscontrol.Menu,
				Resource: menuFromDB.UID,
				Action:   accesscontrol.Write,
			})
		}
		return nil
	}); err != nil {
		return err
	}
	// remove old acl
	if err := srv.authorizeSrv.RemoveResourcePoliciesByCategory(org.ID, accesscontrol.Menu); err != nil {
		return err
	}
	// add new acl
	for _, aclParam := range acl {
		if err := srv.authorizeSrv.AddResourcePolicy(&aclParam); err != nil {
			return err
		}
	}
	return nil
}

// UpdateRolesOfOrgMenus updates roles for current org.
func (srv *menService) UpdateRolesOfOrgMenus(ctx context.Context, menus []model.OrgMenuInfo) error {
	user := util.GetUser(ctx)
	if err := srv.db.Transaction(func(tx dbpkg.DB) error {
		for _, menu := range menus {
			menuFromDB := &model.Menu{}
			if err := tx.Get(menuFromDB, "uid=?", menu.MenuUID); err != nil {
				return err
			}
			if err := tx.UpdateSingle(&model.OrgMenu{},
				"role", menu.Role,
				"org_id=? and menu_id=?", user.Org.ID, menuFromDB.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// update acl role
	for _, menu := range menus {
		if err := srv.authorizeSrv.UpdateResourceRole(&model.ResourceACLParam{
			Role:     menu.Role,
			OrgID:    user.Org.ID,
			Category: accesscontrol.Menu,
			Resource: menu.MenuUID,
			Action:   accesscontrol.Write,
		}); err != nil {
			return err
		}
	}
	return nil
}

// saveMenuTree saves menu tree.
func (srv *menService) saveMenuTree(menus model.Menus, orgID int64) error {
	for i, menu := range menus {
		// root menu, clear parent id
		menu.ParentUID = ""
		menu.Order = i
		if err := srv.saveMenu(menu, orgID); err != nil {
			return err
		}
	}
	return nil
}

// saveMenu saves menu/org menu/acl.
func (srv *menService) saveMenu(menu *model.Menu, orgID int64) error {
	menuFromDB := &model.Menu{}
	if menu.UID == "" {
		// if menu no uid, set it
		menu.UID = uuid.GenerateShortUUID()
	}
	if err := srv.db.Get(menuFromDB, "uid=?", menu.UID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err0 := srv.db.Create(menu); err0 != nil {
				return err0
			}
			menuFromDB = menu
		} else {
			return err
		}
	}

	// init the menu of org.
	exist, err := srv.db.Exist(&model.OrgMenu{}, "org_id=? and menu_id=?", orgID, menuFromDB.ID)
	if err != nil {
		return err
	}
	if !exist {
		if err0 := srv.db.Create(&model.OrgMenu{
			OrgID:  orgID,
			MenuID: menuFromDB.ID,
			Role:   menuFromDB.Role,
		}); err0 != nil {
			return err0
		}
	}

	// init menu acl for org.
	if err := srv.authorizeSrv.AddResourcePolicy(&model.ResourceACLParam{
		Role:     menuFromDB.Role,
		OrgID:    orgID,
		Category: accesscontrol.Menu,
		Resource: menuFromDB.UID,
		Action:   accesscontrol.Write,
	}); err != nil {
		return err
	}

	if len(menu.Children) > 0 {
		for i, child := range menu.Children {
			// set parent id
			child.ParentUID = menuFromDB.UID
			child.Order = i
			if err := srv.saveMenu(child, orgID); err != nil {
				return err
			}
		}
	}
	return nil
}
