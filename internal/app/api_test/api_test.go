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
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"context"
	"fmt"
	"testing"

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
}

func (as *ApiSuite) BeforeEach(t provider.T) {
	var cfg ConfigTest

	t.NewStep("Загрузка конфигурации окружения")
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		t.Fatalf("error read test config: %v", err)
	}

	fmt.Println(cfg.Pg)

	l := &logger.EmptyLogger{}

	t.NewStep("Проверка работы базы данных окружения")
	cfx, err := pgxpool.ParseConfig(cfg.Pg)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}

	as.pgConnection, err = pgxpool.NewWithConfig(context.Background(), cfx)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}

	if err = as.pgConnection.Ping(context.Background()); err != nil {
		l.Fatal("[App] Init - can't check connection to sql with error %s", err)
	}

	t.NewStep("Проверка работы хранилища кэша окружения")
	opt, err := redis.ParseURL(cfg.Redis)
	if err != nil {
		t.Fatalf("error create redis connection: %s", err)
	}
	as.rdsClient = redis.NewClient(opt)

	if err := as.rdsClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("can't check connection to redis with error: %s", err)
	}

	t.NewStep("Инициализация репозиториев")

	// Repository
	as.bannerRepository = bp.NewBannerRepository(as.pgConnection)
	cacheRepository := cr.NewCashRedis(as.rdsClient)

	t.NewStep("Инициализация юзкейсов")
	// Use-cases
	bannerUsecase := bu.NewBannerUsecase(as.bannerRepository)
	cacheManager := cm.NewCacheManager(cacheRepository)
	authService := au.NewAuthUsecase()
	as.authService = authService

	t.NewStep("Инициализация обработчиков запросов")
	// Handlers
	bannerHandlers := bh.NewBannerHandlers(bannerUsecase, cacheManager)
	authHandlers := ah.NewAuthHandlers(as.authService)

	t.NewStep("Инициализация роутера")
	// routes
	as.router, err = v1.NewRouter("/api", app.PrepareRoutes(bannerHandlers, cacheManager, authService, authHandlers),
		config.Release, l, nil)
	if err != nil {
		t.Fatalf("init router error: %s", err)
	}
}

func (as *ApiSuite) AfterEach(t provider.T) {
	_, err := as.pgConnection.Exec(context.Background(), `TRUNCATE banner CASCADE`)
	t.Require().NoError(err)

	t.Require().NoError(as.rdsClient.FlushAll(context.Background()).Err())

	as.pgConnection.Close()
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
