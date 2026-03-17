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
	"sync"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
)

type UserCacheService interface {
	GetSignedUser(ctx context.Context, token string) *model.SignedUser
	StoreSignedUser(ctx context.Context, token string, signedUser *model.SignedUser)
	EvictSignedUser(ctx context.Context)
}

type userCacheService struct {
	cache sync.Map
}

func NewUserCacheService() UserCacheService {
	return &userCacheService{}
}

func (uc *userCacheService) GetSignedUser(ctx context.Context, token string) *model.SignedUser {
	uc.cache.Range(func(key, value any) bool {
		fmt.Printf("user cache key=%v,value=%v,token=%v\n", key, value, token)
		return true
	})
	user, ok := uc.cache.Load(token)
	if !ok {
		return nil
	}
	return user.(*model.SignedUser)
}

func (uc *userCacheService) EvictSignedUser(ctx context.Context) {
	user := util.GetUser(ctx)
	if user != nil {
		uc.cache.Delete(user.Token)
	}
}

func (uc *userCacheService) StoreSignedUser(ctx context.Context, token string, signedUser *model.SignedUser) {
	uc.cache.Store(token, signedUser)
}
