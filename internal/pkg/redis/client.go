package redis

import (
	"context"

	"bannersrv/pkg/logger"

	"github.com/ThCompiler/sdi"
	"github.com/pkg/errors"
	redislib "github.com/redis/go-redis/v9"
)

type Config struct {
	URL string
}

type ProviderDeps struct {
	Config Config
	Logger logger.Interface
}

func NewClient(ctx context.Context, cfg Config, l logger.Interface) (*redislib.Client, error) {
	opt, err := redislib.ParseURL(cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "redis parse config")
	}

	client := redislib.NewClient(opt)

	if err = client.Ping(ctx).Err(); err != nil {
		_ = client.Close() //nolint:errcheck // best-effort cleanup

		return nil, errors.Wrap(err, "check connection to redis")
	}

	l.Info("[App] Init - success check connection to redis")

	return client, nil
}

func NewProvider() sdi.Provider[*redislib.Client, ProviderDeps] {
	return sdi.ProviderFunc(func(ctx context.Context, deps ProviderDeps) (*redislib.Client, error) {
		return NewClient(ctx, deps.Config, deps.Logger)
	})
}
