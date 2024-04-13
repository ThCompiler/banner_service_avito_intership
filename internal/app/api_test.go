//go:build integration

package app

import (
	"bannersrv/external/auth"
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app/config"
	"bannersrv/internal/app/delivery/http/middleware"
	v1 "bannersrv/internal/app/delivery/http/v1"
	"bannersrv/internal/banner"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/banner/delivery/http/v1/models/response"
	"bannersrv/internal/banner/entity"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	cmid "bannersrv/internal/caches/delivery/middleware"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	as.router, err = v1.NewRouter("/api", prepareRoutes(bannerHandlers, cacheManager, authService, authHandlers),
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

const (
	activeBanner      = "activeBanner"
	inactiveBanner    = "inactiveBanner"
	cashedBanner      = "cashedBanner"
	otherCashedBanner = "otherCashedBanner"
)

func (as *ApiSuite) TestGetUserBanner(t provider.T) {
	t.Title("Тестирование апи метода GetUserBanner: GET /user_banner")
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
			FeatureID: (*types.NullableID)(types.NewNullObject[types.ID]()),
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
			FeatureID: (*types.NullableID)(types.NewNullObject[types.ID]()),
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
			FeatureID: (*types.NullableID)(types.NewNullObject[types.ID]()),
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

	t.Run("Попытка получения баннера с некорректными параметрами запроса", func(t provider.T) {
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

	t.Run("Попытка получения баннера с передачей не всех параметров запроса", func(t provider.T) {
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

type createBanner struct {
	Content   json.RawMessage `json:"content"`
	FeatureID types.ID        `json:"feature_id"`
	TagIDs    []types.ID      `json:"tag_ids"`
	IsActive  bool            `json:"is_active"`
}

type BannerID struct {
	BannerID types.ID `json:"banner_id"`
}

func (as *ApiSuite) TestCreateBanner(t provider.T) {
	t.Title("Тестирование апи метода CreateBanner: POST /banner")
	const path = "/api/v1/banner"

	t.Run("Успешное создание баннера", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 1,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		body, err := json.Marshal(bnr)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Post(path).
			Body(string(body)).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusCreated).
			End()

		t.NewStep("Проверка результатов")
		var id BannerID
		resp.JSON(&id)

		_, err = as.bannerRepository.GetBanner(bnr.FeatureID, bnr.TagIDs[0],
			types.NullableObject[uint32]{IsNull: false, Value: 1})
		t.Require().NoError(err)
	})

	t.Run("Попытка создать баннер с существующими feature и tag ids", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 2,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		body, err := json.Marshal(bnr)
		t.Require().NoError(err)

		resp := apitest.New().
			Handler(as.router).
			Post(path).
			Body(string(body)).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusCreated).
			End()

		var id BannerID
		resp.JSON(&id)

		_, err = as.bannerRepository.GetBanner(bnr.FeatureID, bnr.TagIDs[0],
			types.NullableObject[uint32]{IsNull: false, Value: 1})
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Post(path).
			Body(string(body)).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusConflict).
			End()
	})

	t.Run("Попытка создать баннер с неверным телом в запросе", func(t provider.T) {
		t.WithNewStep("Тестирование", func(ctx provider.StepCtx) {
			t.NewStep("Без поля контент")
			apitest.New().
				Handler(as.router).
				Post(path).
				Body(
					`
					{
						"feature_id": 0,
						"is_active": true,
						"tag_ids": [
							0
						]
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusBadRequest).
				End()

			t.NewStep("Без поля фичи")
			apitest.New().
				Handler(as.router).
				Post(path).
				Body(
					`
					{
						"content": {},
						"is_active": true,
						"tag_ids": [
							0
						]
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusBadRequest).
				End()

			t.NewStep("Без поля состояния")
			apitest.New().
				Handler(as.router).
				Post(path).
				Body(
					`
					{
						"content": {},
						"feature_id": 0,
						"tag_ids": [
							0
						]
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusBadRequest).
				End()

			t.NewStep("Без поля тэгов")
			apitest.New().
				Handler(as.router).
				Post(path).
				Body(
					`
					{
						"content": {},
						"feature_id": 0,
						"is_active": true
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusBadRequest).
				End()

			t.NewStep("Неверный типы полей в теле запроса")
			apitest.New().
				Handler(as.router).
				Post(path).
				Body(
					`
					{
						"content": 2,
						"feature_id": "",
						"is_active": 2,
						tag_ids": [
							"0"
						  ]
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusBadRequest).
				End()
		})
	})

	t.Run("Попытка создать баннер неавторизованным пользователем", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Post(path).
			Body(
				`
					{
						"content": {},
						"feature_id": 0,
						"is_active": true,
						tag_ids": [
							0
						  ]
					}
				`).
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка создать баннер пользователя с неверными правами", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Post(path).
			Body(
				`
					{
						"content": {},
						"feature_id": 0,
						"is_active": true,
						tag_ids": [
							0
						  ]
					}
				`).
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})
}

func (as *ApiSuite) TestUpdateBanner(t provider.T) {
	t.Title("Тестирование апи метода UpdateBanner: PATCH /banner")
	const path = "/api/v1/banner"

	t.Run("Успешное обновление баннера", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 2,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(25, []types.ID{10},
			`{}`, true)
		t.Require().NoError(err)

		body, err := json.Marshal(bnr)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/%d", path, bannerID).
			Body(string(body)).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
			FeatureID: (*types.NullableID)(types.NewObject(bnr.FeatureID)),
			TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
		}, 0, 100)
		t.Require().NoError(err)

		t.Require().Len(bnrs, 1)
		t.Require().Equal(bnr.IsActive, bnrs[0].IsActive)
		t.Require().Len(bnrs[0].Versions, 2)
		t.Require().EqualValues(string(bnr.Content), bnrs[0].Versions[1].Content)
	})

	t.Run("Попытка обновить баннер на существующие у другого баннера feature и tag ids", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 3,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		body, err := json.Marshal(bnr)
		t.Require().NoError(err)

		_, err = as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		bannerIDToUpdate, err := as.bannerRepository.CreateBanner(bnr.FeatureID+1, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/%d", path, bannerIDToUpdate).
			Body(string(body)).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusConflict).
			End()
	})

	t.Run("Успешное обновление баннера с одним из полей", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 25,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.WithNewStep("Тестирование только с полем feature_id", func(ctx provider.StepCtx) {
			t.NewStep("Тестирование")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"feature_id": 15
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(types.ID(15))),
				TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().EqualValues(bnr.TagIDs, bnrs[0].TagIDs)
			t.Require().EqualValues(15, bnrs[0].FeatureID)
			t.Require().EqualValues(bnr.IsActive, bnrs[0].IsActive)
			t.Require().Len(bnrs[0].Versions, 1)
			t.Require().EqualValues(string(bnr.Content), bnrs[0].Versions[0].Content)
		})

		t.WithNewStep("Тестирование только с полем is_active", func(ctx provider.StepCtx) {
			t.NewStep("Тестирование")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"is_active": false
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(types.ID(15))),
				TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().EqualValues(bnr.TagIDs, bnrs[0].TagIDs)
			t.Require().EqualValues(15, bnrs[0].FeatureID)
			t.Require().False(bnrs[0].IsActive)
			t.Require().Len(bnrs[0].Versions, 1)
			t.Require().EqualValues(string(bnr.Content), bnrs[0].Versions[0].Content)
		})

		t.WithNewStep("Тестирование только с полем tag_ids", func(ctx provider.StepCtx) {
			t.NewStep("Тестирование")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"tag_ids": [98, 23]
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(types.ID(15))),
				TagID:     (*types.NullableID)(types.NewObject(types.ID(23))),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().EqualValues([]types.ID{98, 23}, bnrs[0].TagIDs)
			t.Require().EqualValues(15, bnrs[0].FeatureID)
			t.Require().False(bnrs[0].IsActive)
			t.Require().Len(bnrs[0].Versions, 1)
			t.Require().EqualValues(string(bnr.Content), bnrs[0].Versions[0].Content)
		})

		t.WithNewStep("Тестирование только с полем tag_ids", func(ctx provider.StepCtx) {
			t.NewStep("Тестирование")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"content": {"title":21}
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(types.ID(15))),
				TagID:     (*types.NullableID)(types.NewObject(types.ID(23))),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().EqualValues([]types.ID{98, 23}, bnrs[0].TagIDs)
			t.Require().EqualValues(15, bnrs[0].FeatureID)
			t.Require().False(bnrs[0].IsActive)
			t.Require().Len(bnrs[0].Versions, 2)
			t.Require().EqualValues(string(bnr.Content), bnrs[0].Versions[0].Content)
			t.Require().EqualValues(types.Content("{\"title\": 21}"), bnrs[0].Versions[1].Content)
		})
	})

	t.Run("Успешное обновление баннеров с сохранением только трёх последних версий", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		const (
			firstContent  = `{"title": 1}`
			secondContent = `{"title": 1}`
			thirdContent  = `{"title": 1}`
			fourthContent = `{"title": 4}`
		)

		bnr := &createBanner{
			Content:   json.RawMessage(firstContent),
			FeatureID: 93,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.WithNewStep("Тестирование", func(ctx provider.StepCtx) {
			t.NewStep("Добавление второй версии")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"content": `+secondContent+`
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err := as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(bnr.FeatureID)),
				TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().Len(bnrs[0].Versions, 2)
			t.Require().EqualValues(secondContent, bnrs[0].Versions[1].Content)

			t.NewStep("Добавление третьей версии")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"content": `+thirdContent+`
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err = as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(bnr.FeatureID)),
				TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().Len(bnrs[0].Versions, 3)
			t.Require().EqualValues(secondContent, bnrs[0].Versions[1].Content)
			t.Require().EqualValues(secondContent, bnrs[0].Versions[2].Content)

			t.NewStep("Добавление четвёртой версии")
			apitest.New().
				Handler(as.router).
				Patchf("%s/%d", path, bannerID).
				Body(
					`
					{
						"content": `+fourthContent+`
					}
				`).
				Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
				Expect(t).
				Status(http.StatusOK).
				End()

			t.NewStep("Проверка результатов")
			bnrs, err = as.bannerRepository.GetBanners(&entity.BannerInfo{
				FeatureID: (*types.NullableID)(types.NewObject(bnr.FeatureID)),
				TagID:     (*types.NullableID)(types.NewObject(bnr.TagIDs[0])),
			}, 0, 100)
			t.Require().NoError(err)

			t.Require().Len(bnrs, 1)
			t.Require().Len(bnrs[0].Versions, 3)
			t.Require().EqualValues(2, bnrs[0].Versions[0].Version)
			t.Require().EqualValues(secondContent, bnrs[0].Versions[0].Content)
			t.Require().EqualValues(thirdContent, bnrs[0].Versions[1].Content)
			t.Require().EqualValues(fourthContent, bnrs[0].Versions[2].Content)
		})

	})

	t.Run("Попытка обновить баннер с неверным типом полей в теле запроса", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 90,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/%d", path, bannerID).
			Body(
				`
					{
						"content": 2,
						"feature_id": "",
						"is_active": 2,
						tag_ids": [
							"0"
						  ]
					}
				`).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка обновить баннер с неверным типом параметра в строке запроса", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/more", path).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка обновить баннер неавторизованным пользователем", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 102,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/%d", path, bannerID).
			Body(
				`
					{
						"content": {},
						"feature_id": 87,
						"is_active": true,
						tag_ids": [
							0
						  ]
					}
				`).
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка обновить баннер пользователя с неверными правами", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")

		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 85,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}

		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Patchf("%s/%d", path, bannerID).
			Body(
				`
					{
						"content": {},
						"feature_id": 20,
						"is_active": true,
						tag_ids": [
							0
						  ]
					}
				`).
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})
}

func (as *ApiSuite) checkDeleted(bannerID types.ID) error {
	id := 0
	return as.pgConnection.
		QueryRow(context.Background(), "SELECT banner_id FROM features_tags_banner WHERE banner_id = $1", bannerID).
		Scan(&id)
}

func (as *ApiSuite) TestDeleteBanner(t provider.T) {
	t.Title("Тестирование апи метода DeleteBanner: DELETE /banner")
	const path = "/api/v1/banner"

	t.Run("Успешное удаление баннера", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 1,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Deletef("%s/%d", path, bannerID).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNoContent).
			End()

		t.NewStep("Проверка результатов")

		t.Require().ErrorIs(as.checkDeleted(bannerID), pgx.ErrNoRows)
	})

	t.Run("Попытка удалить не существующий баннер", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Deletef("%s/100", path).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка удалить баннер с неверным типом параметра в строке запроса", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Deletef("%s/more", path).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка удалить баннер неавторизованным пользователем", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Deletef("%s/%d", path, 2).
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка удалить баннер пользователя с неверными правами", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Deletef("%s/%d", path, 2).
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})
}

const timeWaitCron = 50

func (as *ApiSuite) TestDeleteFilterBanner(t provider.T) {
	t.Title("Тестирование апи метода DeleteFilterBanner: DELETE /filter_banner")
	const path = "/api/v1/filter_banner"

	t.Run("Успешное удаление баннера", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		bnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 1,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.FeatureIDParam, "1").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNoContent).
			End()

		t.NewStep("Проверка результатов")

		<-time.After(timeWaitCron * time.Millisecond)

		t.Require().ErrorIs(as.checkDeleted(bannerID), pgx.ErrNoRows)
	})

	t.Run("Успешное удаление баннера по фиче", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 25,
			TagIDs:    []types.ID{10, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 25,
			TagIDs:    []types.ID{30, 40},
			IsActive:  true,
		}

		firstBannerID, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		secondBannerID, err := as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.FeatureIDParam, "25").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNoContent).
			End()

		t.NewStep("Проверка результатов")

		<-time.After(timeWaitCron * time.Millisecond)

		t.Require().ErrorIs(as.checkDeleted(firstBannerID), pgx.ErrNoRows)
		t.Require().ErrorIs(as.checkDeleted(secondBannerID), pgx.ErrNoRows)
	})

	t.Run("Успешное удаление баннера по тэгу", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 26,
			TagIDs:    []types.ID{45, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title": "banner", "width": 30}`),
			FeatureID: 27,
			TagIDs:    []types.ID{30, 45},
			IsActive:  true,
		}

		firstBannerID, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		secondBannerID, err := as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.TagIDParam, "45").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNoContent).
			End()

		t.NewStep("Проверка результатов")

		<-time.After(timeWaitCron * time.Millisecond)

		t.Require().ErrorIs(as.checkDeleted(firstBannerID), pgx.ErrNoRows)
		t.Require().ErrorIs(as.checkDeleted(secondBannerID), pgx.ErrNoRows)
	})

	t.Run("Попытка удалить не существующий баннер", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.FeatureIDParam, "100").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusNotFound).
			End()
	})

	t.Run("Попытка удалить баннер без параметров запроса", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка удалить баннер с неверным типом параметра в строке запроса", func(t provider.T) {
		t.NewStep("Тестирование неверного типа feature id")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.FeatureIDParam, "mir").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		t.NewStep("Тестирование неверного типа tag id")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Query(bh.FeatureIDParam, "2").Query(bh.TagIDParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка удалить баннер неавторизованным пользователем", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка удалить баннер пользователя с неверными правами", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Delete(path).
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})
}

func (as *ApiSuite) TestGetAdminBanner(t provider.T) {
	t.Title("Тестирование апи метода GetAdminBanner: GET /banner")
	const path = "/api/v1/banner"

	t.Run("Успешное получение списка баннеров", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		bnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 1,
			TagIDs:    []types.ID{2, 4, 3},
			IsActive:  true,
		}
		bannerID, err := as.bannerRepository.CreateBanner(bnr.FeatureID, bnr.TagIDs,
			types.Content(bnr.Content), bnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "1").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 1)
		t.Require().EqualValues(bnrs[0].ID, bannerID)
		t.Require().Len(bnrs[0].Versions, 1)
		t.Require().EqualValues(bnr.Content, bnrs[0].Versions[0].Content)
	})

	t.Run("Успешное получение списка баннеров по фиче", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 25,
			TagIDs:    []types.ID{10, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 25,
			TagIDs:    []types.ID{30, 40},
			IsActive:  true,
		}

		firstBannerID, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		secondBannerID, err := as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "25").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 2)
		t.Require().EqualValues(bnrs[0].ID, firstBannerID)
		t.Require().Len(bnrs[0].Versions, 1)
		t.Require().EqualValues(firstBnr.Content, bnrs[0].Versions[0].Content)

		t.Require().EqualValues(bnrs[1].ID, secondBannerID)
		t.Require().Len(bnrs[1].Versions, 1)
		t.Require().EqualValues(secondBnr.Content, bnrs[1].Versions[0].Content)
	})

	t.Run("Успешное получение списка баннеров по тэгу", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 26,
			TagIDs:    []types.ID{45, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 27,
			TagIDs:    []types.ID{30, 45},
			IsActive:  true,
		}

		firstBannerID, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		secondBannerID, err := as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.TagIDParam, "45").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 2)
		t.Require().EqualValues(bnrs[0].ID, firstBannerID)
		t.Require().Len(bnrs[0].Versions, 1)
		t.Require().EqualValues(firstBnr.Content, bnrs[0].Versions[0].Content)

		t.Require().EqualValues(bnrs[1].ID, secondBannerID)
		t.Require().Len(bnrs[1].Versions, 1)
		t.Require().EqualValues(secondBnr.Content, bnrs[1].Versions[0].Content)
	})

	t.Run("Успешное получение ограниченного списка баннеров по числу", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 31,
			TagIDs:    []types.ID{45, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 31,
			TagIDs:    []types.ID{30, 24},
			IsActive:  true,
		}

		firstBannerID, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		_, err = as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "31").Query(bh.LimitParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 1)
		t.Require().EqualValues(bnrs[0].ID, firstBannerID)
		t.Require().Len(bnrs[0].Versions, 1)
		t.Require().EqualValues(firstBnr.Content, bnrs[0].Versions[0].Content)
	})

	t.Run("Успешное получение смещённого списка баннеров", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 32,
			TagIDs:    []types.ID{45, 25},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 32,
			TagIDs:    []types.ID{30, 24},
			IsActive:  true,
		}

		_, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		secondBannerID, err := as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "32").Query(bh.OffsetParam, "1").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 1)
		t.Require().EqualValues(bnrs[0].ID, secondBannerID)
		t.Require().Len(bnrs[0].Versions, 1)
		t.Require().EqualValues(firstBnr.Content, bnrs[0].Versions[0].Content)
	})

	t.Run("Успешное получение списка баннеров при фильтре запрашивающем несуществующий баннер", func(t provider.T) {
		t.NewStep("Инициализация тестовых данных")
		firstBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 47,
			TagIDs:    []types.ID{46, 13},
			IsActive:  true,
		}
		secondBnr := &createBanner{
			Content:   json.RawMessage(`{"title":"banner","width":30}`),
			FeatureID: 98,
			TagIDs:    []types.ID{23, 46},
			IsActive:  true,
		}

		_, err := as.bannerRepository.CreateBanner(firstBnr.FeatureID, firstBnr.TagIDs,
			types.Content(firstBnr.Content), firstBnr.IsActive)
		t.Require().NoError(err)

		_, err = as.bannerRepository.CreateBanner(secondBnr.FeatureID, secondBnr.TagIDs,
			types.Content(secondBnr.Content), secondBnr.IsActive)
		t.Require().NoError(err)

		t.NewStep("Тестирование")
		resp := apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "100").Query(bh.TagIDParam, "2").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusOK).
			End()

		t.NewStep("Проверка результатов")
		var bnrs []response.Banner

		resp.JSON(&bnrs)

		t.Require().Len(bnrs, 0)
	})

	t.Run("Попытка получить список баннеров с неверным типом параметра в строке запроса", func(t provider.T) {
		t.NewStep("Тестирование неверного типа feature id")
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.FeatureIDParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		t.NewStep("Тестирование неверного типа tag id")
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.TagIDParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		t.NewStep("Тестирование неверного типа limit")
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.LimitParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()

		t.NewStep("Тестирование неверного типа offset")
		apitest.New().
			Handler(as.router).
			Get(path).
			Query(bh.OffsetParam, "mir").
			Header(middleware.TokenHeaderField, string(as.authService.GetAdminToken())).
			Expect(t).
			Status(http.StatusBadRequest).
			End()
	})

	t.Run("Попытка получить список баннеров неавторизованным пользователем", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Get(path).
			Expect(t).
			Status(http.StatusUnauthorized).
			End()
	})

	t.Run("Попытка получить список баннеров пользователя с неверными правами", func(t provider.T) {
		t.NewStep("Тестирование")
		apitest.New().
			Handler(as.router).
			Get(path).
			Header(middleware.TokenHeaderField, string(as.authService.GetUserToken())).
			Expect(t).
			Status(http.StatusForbidden).
			End()
	})
}

func TestRunApiTest(t *testing.T) {
	suite.RunSuite(t, new(ApiSuite))
}