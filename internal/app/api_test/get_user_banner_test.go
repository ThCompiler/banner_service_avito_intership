package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/banner/entity"
	cmid "bannersrv/internal/caches/delivery/middleware"
	"bannersrv/internal/pkg/types"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
