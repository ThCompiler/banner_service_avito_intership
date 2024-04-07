//go:build integration

package app

import (
	"bannersrv/external/auth"
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app/delivery/http/middleware"
	v1 "bannersrv/internal/app/delivery/http/v1"
	"bannersrv/internal/banner"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/banner/entity"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	cmid "bannersrv/internal/caches/delivery/middleware"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jmoiron/sqlx"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"github.com/redis/go-redis/v9"
	"github.com/steinfletcher/apitest"
	"net/http"
	"testing"
)

type ConfigTest struct {
	Pg    string `env:"PG_STRING"`
	Redis string `env:"REDIS_STRING"`
}

type ApiSuite struct {
	suite.Suite
	router           *gin.Engine
	pgConnection     *sqlx.DB
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

	l := &logger.EmptyLogger{}

	t.NewStep("Проверка работы базы данных окружения")
	as.pgConnection, err = sqlx.Open("postgres", cfg.Pg)
	if err != nil {
		t.Fatalf("error create postgres connection: %s", err)
	}

	if err := as.pgConnection.Ping(); err != nil {
		t.Fatalf("can't check connection to sql with error %s", err)
	}

	t.NewStep("Проверка работы хранилища кэша окружения")
	opt, err := redis.ParseURL(cfg.Redis)
	if err != nil {
		t.Fatalf("error create redis connection: %s", err)
	}
	as.rdsClient = redis.NewClient(opt)

	if err := as.rdsClient.Ping(context.Background()).Err(); err != nil {
		l.Fatal("can't check connection to redis with error: %s", err)
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
	as.router, err = v1.NewRouter("/api", l, prepareRoutes(bannerHandlers, cacheManager, authService, authHandlers))
	if err != nil {
		t.Fatalf("init router error: %s", err)
	}
}

func (as *ApiSuite) AfterEach(t provider.T) {
	_, err := as.pgConnection.Exec(`TRUNCATE banner CASCADE`)
	t.Require().NoError(err)

	t.Require().NoError(as.rdsClient.FlushAll(context.Background()).Err())

	t.Require().NoError(as.pgConnection.Close())
}

const (
	activeBanner      = "activeBanner"
	inactiveBanner    = "inactiveBanner"
	cashedBanner      = "cashedBanner"
	otherCashedBanner = "otherCashedBanner"
)

func (as *ApiSuite) TestGetUserBanner(t provider.T) {
	t.Title("Тестирование апи метода GetUserBanner: /user_banner")
	t.NewStep("Инициализация тестовых данных")
	const path = "/api/v1/user_banner"
	const updatedContent = `{"title": "updated_banner", "width": 30}`

	bannerList := map[string]*entity.Banner{
		activeBanner: {
			Content:   `{"title": "banner", "width": 30}`,
			FeatureId: 1,
			TagIds:    []types.Id{2, 4, 3},
			IsActive:  true,
		},
		cashedBanner: {
			Content:   `{"title": "cashed_banner", "width": 30}`,
			FeatureId: 3,
			TagIds:    []types.Id{5, 1, 2},
			IsActive:  true,
		},
		otherCashedBanner: {
			Content:   `{"title": "other_cashed_banner", "width": 30}`,
			FeatureId: 4,
			TagIds:    []types.Id{5, 1, 2},
			IsActive:  true,
		},
		inactiveBanner: {
			Content:   `{"title": "disabled_banner", "width": 30}`,
			FeatureId: 2,
			TagIds:    []types.Id{1, 4, 5},
			IsActive:  false,
		},
	}

	for _, bn := range bannerList {
		id, err := as.bannerRepository.CreateBanner(bn)
		t.Require().NoError(err)
		bn.Id = id
	}

	t.Run("Успешное получение активного баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "1").Query(bh.TagIdParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerList[activeBanner].Content)).
			Status(http.StatusOK).
			End()
	})

	t.Run("Успешное получение активного баннера с обязательным запросом к базе после кэширования", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").Query(bh.TagIdParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerList[cashedBanner].Content)).
			Status(http.StatusOK).
			End()

		_, err := as.bannerRepository.UpdateBanner(&entity.BannerUpdate{
			Id:        bannerList[cashedBanner].Id,
			Content:   types.NewObject[types.Content](updatedContent),
			TagIds:    types.NewNullObject[[]types.Id](),
			FeatureId: types.NewNullObject[types.Id](),
			IsActive:  types.NewNullObject[bool](),
		})
		t.Require().NoError(err)

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").Query(bh.TagIdParam, "1").Query(cmid.UseLastRevisionParam, "true").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(updatedContent).
			Status(http.StatusOK).
			End()
	})

	t.Run("Успешное получение активного баннера из кэша", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "4").Query(bh.TagIdParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerList[otherCashedBanner].Content)).
			Status(http.StatusOK).
			End()

		_, err := as.bannerRepository.UpdateBanner(&entity.BannerUpdate{
			Id:        bannerList[otherCashedBanner].Id,
			Content:   types.NewObject[types.Content](updatedContent),
			TagIds:    types.NewNullObject[[]types.Id](),
			FeatureId: types.NewNullObject[types.Id](),
			IsActive:  types.NewNullObject[bool](),
		})
		t.Require().NoError(err)

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "4").Query(bh.TagIdParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerList[otherCashedBanner].Content)).
			Status(http.StatusOK).
			End()

	})

	t.Run("Попытка получения неактивного баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "2").Query(bh.TagIdParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка получения несуществующего баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").Query(bh.TagIdParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка получения баннера неавторизованным пользователем", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").Query(bh.TagIdParam, "4").
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка получения баннера пользователя с неверными правами", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").Query(bh.TagIdParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})

	t.Run("Попытка передачи некорректных параметров запроса", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "mir").Query(bh.TagIdParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "4").Query(bh.TagIdParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка передачи не всех параметров запроса", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.TagIdParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIdParam, "3").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})
}

func TestRunApiTest(t *testing.T) {
	suite.RunSuite(t, new(ApiSuite))
}
