package api_test

import (
	"bannersrv/internal/app/delivery/http/middleware"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/steinfletcher/apitest"
)

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
