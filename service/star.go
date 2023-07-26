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

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

//go:generate mockgen -source=./star.go -destination=./star_mock.go -package=service

// StarService represents user star manager interface.
type StarService interface {
	// Star stars a resource by uid/type.
	Star(ctx context.Context, resourceUID string, resourceType model.ResourceType) error
	// Unstar unstars a resource by uid/type.
	Unstar(ctx context.Context, resourceUID string, resourceType model.ResourceType) error
	// IsStarred checks if star the resource by uid/type.
	IsStarred(ctx context.Context, resourceUID string, resourceType model.ResourceType) (bool, error)
}

// starService implements StarService interface.
type starService struct {
	db dbpkg.DB
}

// NewStarService creates a StarService instance.
func NewStarService(db dbpkg.DB) StarService {
	return &starService{
		db: db,
	}
}

// Star stars a resource by uid/type.
func (srv *starService) Star(ctx context.Context, resourceUID string, resourceType model.ResourceType) error {
	started, err := srv.IsStarred(ctx, resourceUID, resourceType)
	if err != nil {
		return err
	}
	if started {
		return nil
	}
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	star := &model.Star{
		OrgID:        signedUser.Org.ID,
		UserID:       userID,
		ResourceUID:  resourceUID,
		ResourceType: resourceType,
	}
	star.CreatedBy = userID
	star.UpdatedBy = userID
	return srv.db.Create(star)
}

// Unstar unstars a resource by uid/type.
func (srv *starService) Unstar(ctx context.Context, resourceUID string, resourceType model.ResourceType) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(
		&model.Star{},
		"user_id=? and org_id=? and resource_uid=? and resource_type=?",
		signedUser.User.ID, signedUser.Org.ID, resourceUID, resourceType,
	)
}

// IsStarred checks if star the resource by uid/type.
func (srv *starService) IsStarred(ctx context.Context, resourceUID string, resourceType model.ResourceType) (bool, error) {
	signedUser := util.GetUser(ctx)
	return srv.db.Exist(
		&model.Star{},
		"user_id=? and org_id=? and resource_uid=? and resource_type=?",
		signedUser.User.ID, signedUser.Org.ID, resourceUID, resourceType,
	)
}
