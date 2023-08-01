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

//go:generate mockgen -source=./tag.go -destination=./tag_mock.go -package=service

// TagService represents tag manager service.
type TagService interface {
	// FindTags returns tag list by given term prefix.
	FindTags(ctx context.Context, term string) (tags []string, err error)
	// SaveTags saves tags and tag relation with resource.
	SaveTags(orgID int64, tags []string, resourceUID string, resourceType model.ResourceType) error
}

// tagService implements TagService interface.
type tagService struct {
	db dbpkg.DB
}

// NewTagService creates a TagService instance.
func NewTagService(db dbpkg.DB) TagService {
	return &tagService{
		db: db,
	}
}

// FindTags returns tag list by given term prefix.
func (srv *tagService) FindTags(ctx context.Context, term string) (tags []string, err error) {
	var rs []model.Tag
	signedUser := util.GetUser(ctx)
	if err := srv.db.FindForPaging(&rs, 0, 100, "id desc", "org_id=? and term like ?", signedUser.Org.ID, term+"%"); err != nil {
		return nil, err
	}
	for _, tag := range rs {
		tags = append(tags, tag.Term)
	}
	return tags, nil
}

// SaveTags saves tags and tag relation with resource.
func (srv *tagService) SaveTags(orgID int64, tags []string, resourceUID string, resourceType model.ResourceType) error {
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// check tag if exist, if not exist create them
		var tagList []model.Tag
		if err := tx.Find(&tagList, "org_id=? and term in ?", orgID, tags); err != nil {
			return err
		}
		existTags := make(map[string]struct{})
		for _, tag := range tagList {
			existTags[tag.Term] = struct{}{}
		}
		for _, term := range tags {
			_, exist := existTags[term]
			if !exist {
				createTag := model.Tag{
					OrgID: orgID,
					Term:  term,
				}
				if err := tx.Create(&createTag); err != nil {
					return err
				}
				tagList = append(tagList, createTag)
			}
		}
		// deleve old ralations
		if err := tx.Delete(&model.ResourceTag{},
			"org_id=? and type=? and resource_uid=?", orgID, resourceType, resourceUID); err != nil {
			return err
		}
		// create new relations
		for _, tag := range tagList {
			if err := tx.Create(&model.ResourceTag{
				OrgID:       orgID,
				TagID:       tag.ID,
				ResourceUID: resourceUID,
				Type:        resourceType,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
