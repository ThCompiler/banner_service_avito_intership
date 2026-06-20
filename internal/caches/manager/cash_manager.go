package manager

import (
	"bannersrv/internal/caches"
	"bannersrv/internal/pkg/types"
	"context"
	"fmt"
	"time"

	"github.com/ThCompiler/sdi"
)

const (
	cacheExpiredTime = 5 * time.Minute
)

type CacheManager struct {
	rep caches.Repository
}

func NewProvider() sdi.Provider[caches.Manager, caches.Repository] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, repository caches.Repository) (caches.Manager, error) {
		return &CacheManager{rep: repository}, nil
	})
}

func (cm *CacheManager) HaveCache(featureID, tagID types.ID, version *uint32) (types.Content, error) {
	key := fmt.Sprintf("%d-%d", featureID, tagID)
	if version != nil {
		key = fmt.Sprintf("%s-%d", key, *version)
	}

	return cm.rep.HaveCache(key)
}

func (cm *CacheManager) SetCache(featureID, tagID types.ID, version *uint32, content types.Content) error {
	key := fmt.Sprintf("%d-%d", featureID, tagID)
	if version != nil {
		key = fmt.Sprintf("%s-%d", key, *version)
	}

	return cm.rep.SetCache(key, content, cacheExpiredTime)
}
