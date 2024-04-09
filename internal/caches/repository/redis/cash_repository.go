package redis

import (
	"bannersrv/internal/caches/repository"
	"bannersrv/internal/pkg/types"
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type CashRedis struct {
	client *redis.Client
	ctx    context.Context
}

func NewCashRedis(client *redis.Client) *CashRedis {
	return &CashRedis{client: client, ctx: context.Background()}
}

func (cr *CashRedis) SetCache(key string, content types.Content, ttl time.Duration) error {
	if err := cr.client.Set(cr.ctx, key, string(content), ttl).Err(); err != nil {
		return errors.Wrapf(err,
			"error when try save in cache with key: %s", key)
	}

	return nil
}

func (cr *CashRedis) HaveCache(key string) (types.Content, error) {
	var content string
	if err := cr.client.Get(cr.ctx, key).Scan(&content); err != nil {
		if errors.Is(err, redis.Nil) {
			err = repository.ErrorCacheMiss
		}

		return "", errors.Wrapf(err,
			"error when try get cache with key: %s", key)
	}

	return types.Content(content), nil
}
