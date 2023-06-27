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

//go:generate mockgen -source=./component.go -destination=./component_mock.go -package=service

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
)

// ComponentService represents platform component manager interface.
type ComponentService interface {
	// Initialize initializes supported platform components.
	Initialize() error
	// LoadComponentTree returns supported platform components.
	LoadComponentTree(ctx context.Context) (model.Components, error)
	// CreateComponent creates a new component.
	CreateComponent(ctx context.Context, cmp *model.Component) (string, error)
	// UpdateComponent updates component.
	UpdateComponent(ctx context.Context, cmp *model.Component) error
	// DeleteComponentByUID deletes component than manual created.
	DeleteComponentByUID(ctx context.Context, uid string) error

	// SortComponents sorts component list.
	SortComponents(ctx context.Context, cmps model.Components) error

	// GetComponentTreeByCurrentOrg returns component tree that current user can access of current org.
	GetComponentTreeByCurrentOrg(ctx context.Context) (model.Components, error)
}

// componentService implements ComponentService interface.
type componentService struct {
	db dbpkg.DB

	orgSrv       OrgService
	authorizeSrv AuthorizeService
}

// NewComponentService creates a ComponentService instance.
func NewComponentService(db dbpkg.DB, orgSrv OrgService, authorizeSrv AuthorizeService) ComponentService {
	return &componentService{
		db:           db,
		orgSrv:       orgSrv,
		authorizeSrv: authorizeSrv,
	}
}

// Initialize initializes supported platform components.
func (srv *componentService) Initialize() error {
	mainOrg, err := srv.orgSrv.GetOrgByName(context.TODO(), constant.AdminOrgName)
	if err != nil {
		return err
	}
	var components model.Components
	if err := jsonUnmarshalFn([]byte(linsight.Components), &components); err != nil {
		return err
	}
	return srv.saveComponentTree(components, mainOrg.ID)
}

// LoadComponentTree returns supported platform components.
func (srv *componentService) LoadComponentTree(ctx context.Context) (model.Components, error) {
	var cmps model.Components
	if err := srv.db.Find(&cmps); err != nil {
		return nil, err
	}
	return cmps.ToTree(), nil
}

// UpdateComponent updates component.
func (srv *componentService) UpdateComponent(ctx context.Context, cmp *model.Component) error {
	user := util.GetUser(ctx)
	cmp.UpdatedBy = user.User.ID
	return srv.db.Updates(&model.Component{}, cmp, "uid=?", cmp.UID)
}

// CreateComponent creates a new component.
func (srv *componentService) CreateComponent(ctx context.Context, cmp *model.Component) (string, error) {
	if cmp.ParentUID != "" {
		// check parent if exist
		exist, err := srv.db.Exist(&model.Component{}, "uid=?", cmp.ParentUID)
		if err != nil {
			return "", err
		}
		if !exist {
			return "", errors.New("parent component not exist")
		}
	}
	user := util.GetUser(ctx)
	cmp.UID = uuid.GenerateShortUUID()
	cmp.UpdatedBy = user.User.ID
	cmp.CreatedBy = user.User.ID
	if err := srv.db.Create(&cmp); err != nil {
		return "", err
	}
	return cmp.UID, nil
}

// DeleteComponentByUID deletes component than manual created.
func (srv *componentService) DeleteComponentByUID(ctx context.Context, uid string) error {
	// check cmp if has children
	count, err := srv.db.Count(&model.Component{}, "parent_uid=?", uid)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("component has children")
	}
	return srv.db.Delete(&model.Component{}, "uid=?", uid)
}

// SortComponents sorts component list.
func (srv *componentService) SortComponents(ctx context.Context, cmps model.Components) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for i, cmp := range cmps {
			if err := tx.Updates(model.Component{}, map[string]any{
				"order":      i,
				"parent_uid": cmp.ParentUID,
				"updated_by": user.User.ID,
			}, "uid=?", cmp.UID); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetComponentTreeByCurrentOrg returns component tree that current user can access of current org.
func (srv *componentService) GetComponentTreeByCurrentOrg(ctx context.Context) (model.Components, error) {
	var cmps model.Components
	user := util.GetUser(ctx)
	sql := `select 
		c.uid,c.label,c.path,c.icon,c.component,c.parent_uid,oc.role 
	from components c, org_components oc where c.id=oc.component_id and oc.org_id=?
	`
	if err := srv.db.ExecRaw(&cmps, sql, user.Org.ID); err != nil {
		return nil, err
	}
	var aclParamList []model.ResourceACLParam
	for _, navItem := range cmps {
		aclParamList = append(aclParamList, model.ResourceACLParam{
			Role:     user.Role,
			OrgID:    user.Org.ID,
			Category: accesscontrol.Component,
			Resource: navItem.UID,
			Action:   accesscontrol.Write,
		})
	}
	// check acl
	result, err := srv.authorizeSrv.CheckResourcesACL(aclParamList)
	if err != nil {
		return nil, err
	}
	var accessCmps model.Components
	for i, ok := range result {
		if ok {
			accessCmps = append(accessCmps, cmps[i])
		}
	}
	return accessCmps.ToTree(), nil
}

// saveComponentTree saves component tree.
func (srv *componentService) saveComponentTree(components model.Components, orgID int64) error {
	for i, cmp := range components {
		// root component, clear parent id
		cmp.ParentUID = ""
		cmp.Order = i
		if err := srv.saveComponent(cmp, orgID); err != nil {
			return err
		}
	}
	return nil
}

// saveComponent saves component/org component/acl.
func (srv *componentService) saveComponent(component *model.Component, orgID int64) error {
	cmpFromDB := &model.Component{}
	if component.UID == "" {
		// if component no uid, set it
		component.UID = uuid.GenerateShortUUID()
	}
	if err := srv.db.Get(cmpFromDB, "uid=?", component.UID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err0 := srv.db.Create(component); err0 != nil {
				return err0
			}
			cmpFromDB = component
		} else {
			return err
		}
	}

	// init org's component
	exist, err := srv.db.Exist(&model.OrgComponent{}, "org_id=? and component_id=?", orgID, cmpFromDB.ID)
	if err != nil {
		return err
	}
	if !exist {
		if err0 := srv.db.Create(&model.OrgComponent{
			OrgID:       orgID,
			ComponentID: cmpFromDB.ID,
			Role:        cmpFromDB.Role,
		}); err0 != nil {
			return err0
		}
	}

	// init cmp acl for org.
	if err := srv.authorizeSrv.AddResourcePolicy(&model.ResourceACLParam{
		Role:     cmpFromDB.Role,
		OrgID:    orgID,
		Category: accesscontrol.Component,
		Resource: cmpFromDB.UID,
		Action:   accesscontrol.Write,
	}); err != nil {
		return err
	}

	if len(component.Children) > 0 {
		for i, child := range component.Children {
			// set parent id
			child.ParentUID = cmpFromDB.UID
			child.Order = i
			if err := srv.saveComponent(child, orgID); err != nil {
				return err
			}
		}
	}
	return nil
}
