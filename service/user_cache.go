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
