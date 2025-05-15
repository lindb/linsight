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

//go:generate mockgen -source=./page.go -destination=./page_mock.go -package=service

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
)

// PageService represents platform page manager interface.
type PageService interface {
	// Initialize initializes supported platform pages.
	Initialize() error
	// LoadPageTree returns supported platform pages.
	LoadPageTree(ctx context.Context) (model.Pages, error)
	// CreatePage creates a new page.
	CreatePage(ctx context.Context, page *model.Page) (string, error)
	// UpdatePage updates page.
	UpdatePage(ctx context.Context, page *model.Page) error
	// DeletePageByUID deletes page than manual created.
	DeletePageByUID(ctx context.Context, uid string) error

	// SortPages sorts page list.
	SortPages(ctx context.Context, pages model.Pages) error

	// GetPageTreeByCurrentOrg returns page tree that current user can access of current org.
	GetPageTreeByCurrentOrg(ctx context.Context) (model.Pages, error)
	// GetOrgPages returns page list for org.
	GetOrgPages(ctx context.Context, orgUID string) ([]model.OrgPageInfo, error)
	// SaveOrgPages creates or removes pages for org's pages.
	SaveOrgPages(ctx context.Context, orgUID string, pages []model.OrgPageInfo) error
	// UpdateRolesOfOrgPages updates roles for current org.
	UpdateRolesOfOrgPages(ctx context.Context, pages []model.OrgPageInfo) error
}

// pageService implements PageService interface.
type pageService struct {
	db dbpkg.DB

	orgSrv       OrgService
	authorizeSrv AuthorizeService
}

// NewPageService creates a PageService instance.
func NewPageService(db dbpkg.DB, orgSrv OrgService, authorizeSrv AuthorizeService) PageService {
	return &pageService{
		db:           db,
		orgSrv:       orgSrv,
		authorizeSrv: authorizeSrv,
	}
}

// Initialize initializes supported platform pages.
func (srv *pageService) Initialize() error {
	mainOrg, err := srv.orgSrv.GetOrgByName(context.TODO(), constant.AdminOrgName)
	if err != nil {
		return err
	}
	var pages model.Pages
	if err := jsonUnmarshalFn([]byte(linsight.Pages), &pages); err != nil {
		return err
	}
	return srv.savePageTree(pages, mainOrg.ID)
}

// LoadPageTree returns supported platform pages.
func (srv *pageService) LoadPageTree(ctx context.Context) (model.Pages, error) {
	var pages model.Pages
	if err := srv.db.Find(&pages); err != nil {
		return nil, err
	}
	return pages.ToTree(), nil
}

// UpdatePage updates page.
func (srv *pageService) UpdatePage(ctx context.Context, page *model.Page) error {
	user := util.GetUser(ctx)
	return srv.db.Updates(&model.Page{}, map[string]any{
		"label":       page.Label,
		"path":        page.Path,
		"icon":        page.Icon,
		"role":        page.Role,
		"page":        page.Page,
		"page_config": page.PageConfig,
		"parent_uid":  page.ParentUID,
		"updated_by":  user.User.ID,
	}, "uid=?", page.UID)
}

// CreatePage creates a new page.
func (srv *pageService) CreatePage(ctx context.Context, page *model.Page) (string, error) {
	if page.ParentUID != "" {
		// check parent if exist
		exist, err := srv.db.Exist(&model.Page{}, "uid=?", page.ParentUID)
		if err != nil {
			return "", err
		}
		if !exist {
			return "", errors.New("parent page not exist")
		}
	}
	user := util.GetUser(ctx)
	page.UID = uuid.GenerateShortUUID()
	page.UpdatedBy = user.User.ID
	page.CreatedBy = user.User.ID
	if err := srv.db.Create(&page); err != nil {
		return "", err
	}
	return page.UID, nil
}

// DeletePageByUID deletes page than manual created.
func (srv *pageService) DeletePageByUID(ctx context.Context, uid string) error {
	// check page if has children
	count, err := srv.db.Count(&model.Page{}, "parent_uid=?", uid)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("page has children")
	}
	return srv.db.Delete(&model.Page{}, "uid=?", uid)
}

// SortPages sorts page list.
func (srv *pageService) SortPages(ctx context.Context, pages model.Pages) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for i, page := range pages {
			if err := tx.Updates(model.Page{}, map[string]any{
				"order":      i,
				"parent_uid": page.ParentUID,
				"updated_by": user.User.ID,
			}, "uid=?", page.UID); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPageTreeByCurrentOrg returns page tree that current user can access of current org.
func (srv *pageService) GetPageTreeByCurrentOrg(ctx context.Context) (model.Pages, error) {
	var pages model.Pages
	user := util.GetUser(ctx)
	sql := `select  
		c.uid,c.label,c.path,c.icon,c.page,c.page_config,c.parent_uid,oc.role,c.'order' 
	from pages c, org_pages oc where c.id=oc.page_id and oc.org_id=?
	`
	if err := srv.db.ExecRaw(&pages, sql, user.Org.ID); err != nil {
		return nil, err
	}
	var aclParamList []model.ResourceACLParam
	for _, navItem := range pages {
		aclParamList = append(aclParamList, model.ResourceACLParam{
			Role:     user.Role,
			OrgID:    user.Org.ID,
			Category: accesscontrol.Page,
			Resource: navItem.UID,
			Action:   accesscontrol.Write,
		})
	}
	// check acl
	result, err := srv.authorizeSrv.CheckResourcesACL(aclParamList)
	if err != nil {
		return nil, err
	}
	var accessPages model.Pages
	for i, ok := range result {
		if ok {
			accessPages = append(accessPages, pages[i])
		}
	}
	return accessPages.ToTree(), nil
}

// GetOrgPages returns page list for org.
func (srv *pageService) GetOrgPages(ctx context.Context, orgUID string) ([]model.OrgPageInfo, error) {
	var pages []model.OrgPageInfo
	sql := `select 
		c.uid as page_uid,oc.role 
	from pages c, org_pages oc,orgs o 
	where c.page <> '' and c.id=oc.page_id and oc.org_id=o.id and o.uid=?
	`
	if err := srv.db.ExecRaw(&pages, sql, orgUID); err != nil {
		return nil, err
	}
	return pages, nil
}

// SaveOrgPages creates or removes pages for org's pages.
func (srv *pageService) SaveOrgPages(ctx context.Context, orgUID string, pages []model.OrgPageInfo) error {
	user := util.GetUser(ctx)
	org, err := srv.orgSrv.GetOrgByUID(ctx, orgUID)
	if err != nil {
		return err
	}
	var acl []model.ResourceACLParam
	if err := srv.db.Transaction(func(tx dbpkg.DB) error {
		// remove pages
		if err := tx.Delete(&model.OrgPage{}, "org_id=?", org.ID); err != nil {
			return err
		}
		// create pages
		for _, page := range pages {
			pageFromDB := &model.Page{}
			if err := tx.Get(pageFromDB, "uid=?", page.PageUID); err != nil {
				return err
			}
			if err := tx.Create(&model.OrgPage{
				OrgID:  org.ID,
				PageID: pageFromDB.ID,
				Role:   page.Role,
				BaseModel: model.BaseModel{
					CreatedBy: user.User.ID,
					UpdatedBy: user.User.ID,
				},
			}); err != nil {
				return err
			}
			acl = append(acl, model.ResourceACLParam{
				Role:     pageFromDB.Role,
				OrgID:    org.ID,
				Category: accesscontrol.Page,
				Resource: pageFromDB.UID,
				Action:   accesscontrol.Write,
			})
		}
		return nil
	}); err != nil {
		return err
	}
	// remove old acl
	if err := srv.authorizeSrv.RemoveResourcePoliciesByCategory(org.ID, accesscontrol.Page); err != nil {
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

// UpdateRolesOfOrgPages updates roles for current org.
func (srv *pageService) UpdateRolesOfOrgPages(ctx context.Context, pages []model.OrgPageInfo) error {
	user := util.GetUser(ctx)
	if err := srv.db.Transaction(func(tx dbpkg.DB) error {
		for _, page := range pages {
			pageFromDB := &model.Page{}
			if err := tx.Get(pageFromDB, "uid=?", page.PageUID); err != nil {
				return err
			}
			if err := tx.UpdateSingle(&model.OrgPage{},
				"role", page.Role,
				"org_id=? and page_id=?", user.Org.ID, pageFromDB.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// update acl role
	for _, page := range pages {
		if err := srv.authorizeSrv.UpdateResourceRole(&model.ResourceACLParam{
			Role:     page.Role,
			OrgID:    user.Org.ID,
			Category: accesscontrol.Page,
			Resource: page.PageUID,
			Action:   accesscontrol.Write,
		}); err != nil {
			return err
		}
	}
	return nil
}

// savePageTree saves page tree.
func (srv *pageService) savePageTree(pages model.Pages, orgID int64) error {
	for i, page := range pages {
		// root page, clear parent id
		page.ParentUID = ""
		page.Order = i
		if err := srv.savePage(page, orgID); err != nil {
			return err
		}
	}
	return nil
}

// savePage saves page/org page/acl.
func (srv *pageService) savePage(page *model.Page, orgID int64) error {
	pageFromDB := &model.Page{}
	if page.UID == "" {
		// if page no uid, set it
		page.UID = uuid.GenerateShortUUID()
	}
	if err := srv.db.Get(pageFromDB, "uid=?", page.UID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err0 := srv.db.Create(page); err0 != nil {
				return err0
			}
			pageFromDB = page
		} else {
			return err
		}
	}

	// init org's page
	exist, err := srv.db.Exist(&model.OrgPage{}, "org_id=? and page_id=?", orgID, pageFromDB.ID)
	if err != nil {
		return err
	}
	if !exist {
		if err0 := srv.db.Create(&model.OrgPage{
			OrgID:  orgID,
			PageID: pageFromDB.ID,
			Role:   pageFromDB.Role,
		}); err0 != nil {
			return err0
		}
	}

	// init page acl for org.
	if err := srv.authorizeSrv.AddResourcePolicy(&model.ResourceACLParam{
		Role:     pageFromDB.Role,
		OrgID:    orgID,
		Category: accesscontrol.Page,
		Resource: pageFromDB.UID,
		Action:   accesscontrol.Write,
	}); err != nil {
		return err
	}

	if len(page.Children) > 0 {
		for i, child := range page.Children {
			// set parent id
			child.ParentUID = pageFromDB.UID
			child.Order = i
			if err := srv.savePage(child, orgID); err != nil {
				return err
			}
		}
	}
	return nil
}
