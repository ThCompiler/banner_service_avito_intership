//go:build integration

package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
