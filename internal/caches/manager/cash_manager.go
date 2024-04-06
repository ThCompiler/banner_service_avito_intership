package manager

import (
	"bannersrv/internal/caches"
	"bannersrv/internal/pkg/types"
	"fmt"
	"time"
)

const (
	cacheExpiredTime = 5 * time.Minute
)

type CacheManager struct {
	rep caches.Repository
}

func NewCacheManager(cache caches.Repository) *CacheManager {
	return &CacheManager{
		rep: cache,
	}
}

func (cm *CacheManager) HaveCache(featureId types.Id, tagId types.Id) (types.Content, error) {
	key := fmt.Sprintf("%d-%d", featureId, tagId)
	return cm.rep.HaveCache(key)
}

func (cm *CacheManager) SetCache(featureId types.Id, tagId types.Id, content types.Content) error {
	key := fmt.Sprintf("%d-%d", featureId, tagId)
	return cm.rep.SetCache(key, content, cacheExpiredTime)
}
