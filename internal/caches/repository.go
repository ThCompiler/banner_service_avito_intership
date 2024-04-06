package caches

import (
	"bannersrv/internal/pkg/types"
	"time"
)

type Repository interface {
	HaveCache(key string) (types.Content, error)
	SetCache(key string, content types.Content, ttl time.Duration) error
}
