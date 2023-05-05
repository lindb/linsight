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
	// Star stars a entity by id/type.
	Star(ctx context.Context, entityID int64, entityType model.EntityType) error
	// Unstar unstars a entity by id/type.
	Unstar(ctx context.Context, entityID int64, entityType model.EntityType) error
	// IsStarred checks if star the entity by id/type.
	IsStarred(ctx context.Context, entityID int64, entityType model.EntityType) (bool, error)
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

// Star stars a entity by id/type.
func (srv *starService) Star(ctx context.Context, entityID int64, entityType model.EntityType) error {
	started, err := srv.IsStarred(ctx, entityID, entityType)
	if err != nil {
		return err
	}
	if started {
		return nil
	}
	signedUser := util.GetUser(ctx)
	userID := signedUser.User.ID
	star := &model.Star{
		OrgID:      signedUser.Org.ID,
		UserID:     userID,
		EntityID:   entityID,
		EntityType: entityType,
	}
	star.CreatedBy = userID
	star.UpdatedBy = userID
	return srv.db.Create(star)
}

// Unstar unstars a entity by id/type.
func (srv *starService) Unstar(ctx context.Context, entityID int64, entityType model.EntityType) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(
		&model.Star{},
		"user_id=? and org_id=? and entity_id=? and entity_type=?",
		signedUser.User.ID, signedUser.Org.ID, entityID, entityType,
	)
}

// IsStarred checks if star the entity by id/type.
func (srv *starService) IsStarred(ctx context.Context, entityID int64, entityType model.EntityType) (bool, error) {
	signedUser := util.GetUser(ctx)
	return srv.db.Exist(
		&model.Star{},
		"user_id=? and org_id=? and entity_id=? and entity_type=?",
		signedUser.User.ID, signedUser.Org.ID, entityID, entityType,
	)
}
