//go:build integration

package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/banner/delivery/http/v1/models/response"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
