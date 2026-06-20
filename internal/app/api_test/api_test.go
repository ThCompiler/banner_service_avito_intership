//go:build integration

package api_test

import (
	"bannersrv/external/auth"
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app"
	"bannersrv/internal/app/config"
	v1 "bannersrv/internal/app/delivery/http/v1"
	"bannersrv/internal/banner"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	"bannersrv/internal/caches"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/pg"
	redisclient "bannersrv/internal/pkg/redis"
	"bannersrv/internal/pkg/types"
	"bannersrv/internal/token"
	"bannersrv/pkg/logger"
	"context"
	"testing"

	"github.com/ThCompiler/sdi"
	"github.com/gin-gonic/gin"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"github.com/redis/go-redis/v9"
)

type ConfigTest struct {
	Pg    string `env:"PG_STRING"`
	Redis string `env:"REDIS_STRING"`
}

type ApiSuite struct {
	suite.Suite
	router           *gin.Engine
	pgConnection     *pgxpool.Pool
	rdsClient        *redis.Client
	bannerRepository banner.Repository
	authService      auth.Usecase
	cleanup          func() error
}

type testRuntime struct {
	Router           *gin.Engine
	PGConnection     *pgxpool.Pool
	RedisClient      *redis.Client
	BannerRepository banner.Repository
	AuthService      auth.Usecase
}

type testRuntimeDeps struct {
	Router           *gin.Engine
	PGConnection     *pgxpool.Pool
	RedisClient      *redis.Client
	BannerRepository banner.Repository
	AuthService      auth.Usecase
}

type configDeps struct {
	Config *config.Config
}

type routerDeps struct {
	BannerHandlers *bh.BannerHandlers
	CacheManager   caches.Manager
	TokenService   token.Service
	AuthHandlers   *ah.AuthHandlers
	Logger         logger.Interface
}

func (as *ApiSuite) BeforeEach(t provider.T) {
	var cfg ConfigTest

	t.NewStep("Загрузка конфигурации окружения")
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		t.Fatalf("error read test config: %v", err)
	}

	if cfg.Pg == "" {
		t.Fatal("env PG_STRING is required for integration tests")
	}

	if cfg.Redis == "" {
		t.Fatal("env REDIS_STRING is required for integration tests")
	}

	t.NewStep("Инициализация тестового контейнера")
	testConfig := &config.Config{
		Postgres: config.PG{
			URL:                cfg.Pg,
			MaxConnections:     5,
			MinConnections:     2,
			TTLIDleConnections: 10,
		},
		Redis:    config.Redis{URL: cfg.Redis},
		Mode:     config.Release,
	}

	builder := sdi.NewBuilder()

	for _, err = range []error{
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, _ struct{}) (*config.Config, error) {
				return testConfig, nil
			},
		)),
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, _ struct{}) (logger.Interface, error) {
				return &logger.EmptyLogger{}, nil
			},
		)),
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, deps configDeps) (pg.Config, error) {
				return pg.Config{
					URL:                deps.Config.Postgres.URL,
					MaxConnections:     deps.Config.Postgres.MaxConnections,
					MinConnections:     deps.Config.Postgres.MinConnections,
					TTLIDleConnections: deps.Config.Postgres.TTLIDleConnections,
				}, nil
			},
		)),
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, deps configDeps) (redisclient.Config, error) {
				return redisclient.Config{URL: deps.Config.Redis.URL}, nil
			},
		)),
		sdi.AddProvider(builder, pg.NewProvider()),
		sdi.AddProvider(builder, redisclient.NewProvider()),
		sdi.AddProvider(builder, bp.NewProvider()),
		sdi.AddProvider(builder, cr.NewProvider()),
		sdi.AddProvider(builder, bu.NewProvider()),
		sdi.AddProvider(builder, cm.NewProvider()),
		sdi.AddProvider(builder, au.NewProvider()),
		sdi.AddProvider(builder, bh.NewProvider()),
		sdi.AddProvider(builder, ah.NewProvider()),
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, deps routerDeps) (*gin.Engine, error) {
				routes := app.PrepareRoutes(deps.BannerHandlers, deps.CacheManager, deps.TokenService, deps.AuthHandlers)

				return v1.NewRouter("/api", routes, config.Release, deps.Logger, nil)
			},
		)),
		sdi.AddProvider(builder, sdi.ProviderFuncNoClean(
			func(_ context.Context, deps testRuntimeDeps) (*testRuntime, error) {
				return &testRuntime{
					Router:           deps.Router,
					PGConnection:     deps.PGConnection,
					RedisClient:      deps.RedisClient,
					BannerRepository: deps.BannerRepository,
					AuthService:      deps.AuthService,
				}, nil
			},
		)),
	} {
		if err != nil {
			t.Fatalf("init test container error: %s", err)
		}
	}

	runtime, cleanup, err := sdi.BuildInstance[*testRuntime](context.Background(), builder)
	if err != nil {
		t.Fatalf("build test container error: %s", err)
	}

	as.router = runtime.Router
	as.pgConnection = runtime.PGConnection
	as.rdsClient = runtime.RedisClient
	as.bannerRepository = runtime.BannerRepository
	as.authService = runtime.AuthService
	as.cleanup = cleanup
}

func (as *ApiSuite) AfterEach(t provider.T) {
	if as.pgConnection == nil || as.rdsClient == nil || as.cleanup == nil {
		return
	}

	_, err := as.pgConnection.Exec(context.Background(), `TRUNCATE banner CASCADE`)
	t.Require().NoError(err)

	t.Require().NoError(as.rdsClient.FlushAll(context.Background()).Err())

	t.Require().NoError(as.cleanup())
}

func (as *ApiSuite) checkDeleted(bannerID types.ID) error {
	id := 0
	return as.pgConnection.
		QueryRow(context.Background(), "SELECT banner_id FROM features_tags_banner WHERE banner_id = $1", bannerID).
		Scan(&id)
}

func TestRunApiTest(t *testing.T) {
	suite.RunSuite(t, new(ApiSuite))
}
