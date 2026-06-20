package app

import (
	"context"

	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app/config"
	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	"bannersrv/internal/caches"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/metrics"
	metricsprom "bannersrv/internal/pkg/metrics/prometheus"
	"bannersrv/internal/pkg/pg"
	redisclient "bannersrv/internal/pkg/redis"
	"bannersrv/internal/token"
	"bannersrv/pkg/logger"
	"bannersrv/pkg/server"

	"github.com/ThCompiler/sdi"
	"github.com/gin-gonic/gin"
)

const metricsServiceName = "main"

type Runtime struct {
	Server *server.Server
	Logger logger.Interface
	Config *config.Config
}

type routerDeps struct {
	Config         *config.Config
	BannerHandlers *bh.BannerHandlers
	Cache          caches.Manager
	TokenService   token.Service
	AuthHandlers   *ah.AuthHandlers
	Logger         logger.Interface
	Metrics        metrics.Manager
}

type configDeps struct {
	Config *config.Config
}

type runtimeDeps struct {
	Server *server.Server
	Logger logger.Interface
	Config *config.Config
}

func BuildRuntime(ctx context.Context, configPath string) (*Runtime, func() error, error) {
	builder := sdi.NewBuilder()

	for _, err := range []error{
		sdi.AddProvider(builder, newConfigPathProvider(configPath)),
		sdi.AddProvider(builder, config.NewProvider()),
		sdi.AddProvider(builder, newLoggerParamsProvider()),
		sdi.AddProvider(builder, newPostgresConfigProvider()),
		sdi.AddProvider(builder, newRedisConfigProvider()),
		sdi.AddProvider(builder, newMetricsConfigProvider()),
		sdi.AddProvider(builder, newServerConfigProvider()),
		sdi.AddProvider(builder, logger.NewProvider()),
		sdi.AddProvider(builder, pg.NewProvider()),
		sdi.AddProvider(builder, redisclient.NewProvider()),
		sdi.AddProvider(builder, metricsprom.NewProvider()),
		sdi.AddProvider(builder, bp.NewProvider()),
		sdi.AddProvider(builder, cr.NewProvider()),
		sdi.AddProvider(builder, bu.NewProvider()),
		sdi.AddProvider(builder, cm.NewProvider()),
		sdi.AddProvider(builder, au.NewProvider()),
		sdi.AddProvider(builder, bh.NewProvider()),
		sdi.AddProvider(builder, ah.NewProvider()),
		sdi.AddProvider(builder, newRouterProvider()),
		sdi.AddProvider(builder, server.NewProvider()),
		sdi.AddProvider(builder, newRuntimeProvider()),
	} {
		if err != nil {
			return nil, nil, err
		}
	}

	return sdi.BuildInstance[*Runtime](ctx, builder)
}

func newConfigPathProvider(configPath string) sdi.Provider[config.Path, struct{}] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, _ struct{}) (config.Path, error) {
		return config.Path(configPath), nil
	})
}

func newLoggerParamsProvider() sdi.Provider[logger.Params, configDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps configDeps) (logger.Params, error) {
		return logger.Params{
			AppName:                  deps.Config.LoggerInfo.AppName,
			LogDir:                   deps.Config.LoggerInfo.Directory,
			Level:                    deps.Config.LoggerInfo.Level,
			UseStdAndFile:            deps.Config.LoggerInfo.UseStdAndFile,
			AddLowPriorityLevelToCmd: deps.Config.LoggerInfo.AllowShowLowLevel,
		}, nil
	})
}

func newPostgresConfigProvider() sdi.Provider[pg.Config, configDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps configDeps) (pg.Config, error) {
		return pg.Config{
			URL:                deps.Config.Postgres.URL,
			MaxConnections:     deps.Config.Postgres.MaxConnections,
			MinConnections:     deps.Config.Postgres.MinConnections,
			TTLIDleConnections: deps.Config.Postgres.TTLIDleConnections,
		}, nil
	})
}

func newRedisConfigProvider() sdi.Provider[redisclient.Config, configDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps configDeps) (redisclient.Config, error) {
		return redisclient.Config{URL: deps.Config.Redis.URL}, nil
	})
}

func newMetricsConfigProvider() sdi.Provider[metricsprom.Config, struct{}] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, _ struct{}) (metricsprom.Config, error) {
		return metricsprom.Config{ServiceName: metricsServiceName}, nil
	})
}

func newServerConfigProvider() sdi.Provider[server.Config, configDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps configDeps) (server.Config, error) {
		return server.Config{Port: deps.Config.Port}, nil
	})
}

func newRouterProvider() sdi.Provider[*gin.Engine, routerDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps routerDeps) (*gin.Engine, error) {
		routes := PrepareRoutes(deps.BannerHandlers, deps.Cache, deps.TokenService, deps.AuthHandlers)

		return v1.NewRouter("/api", routes, deps.Config.Mode, deps.Logger, deps.Metrics)
	})
}

func newRuntimeProvider() sdi.Provider[*Runtime, runtimeDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps runtimeDeps) (*Runtime, error) {
		return &Runtime{Server: deps.Server, Logger: deps.Logger, Config: deps.Config}, nil
	})
}
