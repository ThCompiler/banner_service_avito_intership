//go:build integration

package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
