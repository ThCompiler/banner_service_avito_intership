package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
