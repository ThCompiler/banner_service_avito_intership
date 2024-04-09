//go:build integration

package app

import (
	"bannersrv/internal/app/config"
	"context"
	"fmt"
	"net/http"
	"testing"

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

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron/v2"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jmoiron/sqlx"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"github.com/redis/go-redis/v9"
	"github.com/steinfletcher/apitest"

	_ "github.com/jackc/pgx/v5/stdlib"
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

	fmt.Println(cfg.Pg)

	l := &logger.EmptyLogger{}

	t.NewStep("Проверка работы базы данных окружения")
	as.pgConnection, err = sqlx.Open("pgx", cfg.Pg)
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
		t.Fatalf("can't check connection to redis with error: %s", err)
	}

	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		t.Fatal(fmt.Errorf("start cronScheduler error: %s", err))
	}

	t.NewStep("Инициализация репозиториев")

	// Repository
	as.bannerRepository, err = bp.NewBannerRepository(as.pgConnection, cronScheduler, l)
	if err != nil {
		t.Fatal(fmt.Errorf("initialize BannerRepository error: %s", err))
	}
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
	as.router, err = v1.NewRouter("/api", prepareRoutes(bannerHandlers, cacheManager, authService, authHandlers),
		config.Release, l, nil)
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

	bannerContentList := map[string]types.Content{
		activeBanner:      `{"title": "banner", "width": 30}`,
		cashedBanner:      `{"title": "cashed_banner", "width": 30}`,
		otherCashedBanner: `{"title": "other_cashed_banner", "width": 30}`,
		inactiveBanner:    `{"title": "disabled_banner", "width": 30}`,
	}

	bannerList := map[string]*entity.Banner{
		activeBanner: {
			FeatureID: 1,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		},
		cashedBanner: {
			FeatureID: 3,
			TagIDs:    []types.ID{5, 1, 2},
			IsActive:  true,
		},
		otherCashedBanner: {
			FeatureID: 4,
			TagIDs:    []types.ID{5, 1, 2},
			IsActive:  true,
		},
		inactiveBanner: {
			FeatureID: 2,
			TagIDs:    []types.ID{1, 4, 5},
			IsActive:  false,
		},
	}

	for key, bn := range bannerList {
		id, err := as.bannerRepository.CreateBanner(bn.FeatureID, bn.TagIDs, bannerContentList[key], bn.IsActive)
		t.Require().NoError(err)
		bn.ID = id
	}

	t.Run("Успешное получение активного баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "1").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerContentList[activeBanner])).
			Status(http.StatusOK).
			End()
	})

	t.Run("Успешное получение активного баннера с обязательным запросом к базе после кэширования", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerContentList[cashedBanner])).
			Status(http.StatusOK).
			End()

		_, err := as.bannerRepository.UpdateBanner(&entity.BannerUpdate{
			ID:        bannerList[cashedBanner].ID,
			Content:   types.NewObject[types.Content](updatedContent),
			TagIDs:    types.NewNullObject[[]types.ID](),
			FeatureID: types.NewNullObject[types.ID](),
			IsActive:  types.NewNullObject[bool](),
		})
		t.Require().NoError(err)

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "1").Query(cmid.UseLastRevisionParam, "true").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(updatedContent).
			Status(http.StatusOK).
			End()
	})

	t.Run("Успешное получение активного баннера с указанной версией", func(t provider.T) {
		_, err := as.bannerRepository.UpdateBanner(&entity.BannerUpdate{
			ID:        bannerList[cashedBanner].ID,
			Content:   types.NewObject[types.Content](updatedContent),
			TagIDs:    types.NewNullObject[[]types.ID](),
			FeatureID: types.NewNullObject[types.ID](),
			IsActive:  types.NewNullObject[bool](),
		})
		t.Require().NoError(err)

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "1").
			Query(bh.VersionParam, "1").Query(cmid.UseLastRevisionParam, "true").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerContentList[cashedBanner])).
			Status(http.StatusOK).
			End()
	})

	t.Run("Успешное получение активного баннера из кэша", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "4").Query(bh.TagIDParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerContentList[otherCashedBanner])).
			Status(http.StatusOK).
			End()

		_, err := as.bannerRepository.UpdateBanner(&entity.BannerUpdate{
			ID:        bannerList[otherCashedBanner].ID,
			Content:   types.NewObject[types.Content](updatedContent),
			TagIDs:    types.NewNullObject[[]types.ID](),
			FeatureID: types.NewNullObject[types.ID](),
			IsActive:  types.NewNullObject[bool](),
		})
		t.Require().NoError(err)

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "4").Query(bh.TagIDParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Body(string(bannerContentList[otherCashedBanner])).
			Status(http.StatusOK).
			End()
	})

	t.Run("Попытка получения неактивного баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "2").Query(bh.TagIDParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка получения несуществующего баннера", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка получения баннера неавторизованным пользователем", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "4").
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка получения баннера пользователя с неверными правами", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").Query(bh.TagIDParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})

	t.Run("Попытка передачи некорректных параметров запроса", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "mir").Query(bh.TagIDParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "4").Query(bh.TagIDParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка передачи не всех параметров запроса", func(t provider.T) {
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.TagIDParam, "4").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "3").
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})
}

func TestRunApiTest(t *testing.T) {
	suite.RunSuite(t, new(ApiSuite))
}
