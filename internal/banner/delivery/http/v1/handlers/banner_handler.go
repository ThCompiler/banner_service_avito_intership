package handlers

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/banner"
	"bannersrv/internal/banner/delivery/http/v1/models/request"
	"bannersrv/internal/banner/delivery/http/v1/models/response"
	"bannersrv/internal/banner/models"
	"bannersrv/internal/caches"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/slices"
	"net/http"
	"strconv"

	br "bannersrv/internal/banner/repository"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const BannerIDField = "id"

const (
	TagIDParam     = "tag_id"
	FeatureIDParam = "feature_id"
	VersionParam   = "version"
	limitParam     = "limit"
	offsetParam    = "offset"
)

type BannerHandlers struct {
	usecase banner.Usecase
	cache   caches.Manager
}

func NewBannerHandlers(usecase banner.Usecase, cache caches.Manager) *BannerHandlers {
	return &BannerHandlers{usecase: usecase, cache: cache}
}

// CreateBanner
//
//	@Summary		Создание нового баннера.
//	@Description	Добавляет баннер включая его содержания, id фичи, список id тэгов и состояние.
//	@Tags			banner
//	@Accept			json
//	@Param			request	body	request.CreateBanner	true	"Информация о добавляемом пользователе"
//	@Produce		json
//	@Success		201	{object}	response.BannerID	"Баннер успешно добавлен в систему"
//	@Failure		400	{object}	tools.Error			"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		409	"Баннер с указанной парой id фичи и ia тэга уже существует"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/banner [post]
//
//	@Security		AdminToken
func (bh *BannerHandlers) CreateBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	// Получение значения тела запроса
	var createBanner request.CreateBanner
	if code, err := tools.ParseRequestBody(c.Request.Body, &createBanner, request.ValidateCreateBanner, l); err != nil {
		tools.SendError(c, err, code, l)

		return
	}

	createdID, err := bh.usecase.CreateBanner(createBanner.TagsIDs, createBanner.FeatureID,
		createBanner.Content, createBanner.IsActive)
	if err != nil {
		if errors.Is(err, br.ErrorBannerConflictExists) {
			tools.SendErrorStatus(c, err, http.StatusConflict, l)

			return
		}

		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't create banner"))

		return
	}

	tools.SendStatus(c, http.StatusCreated, &response.BannerID{BannerID: createdID}, l)
}

// DeleteBanner
//
//	@Summary		Удаление банера.
//	@Description	Удаляет информацию о банере по его id.
//	@Tags			banner
//	@Param			id	path	integer	true	"Идентификатор баннера"
//	@Produce		json
//	@Success		204	"Баннер успешно удалён"
//	@Failure		400	{object}	tools.Error	"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		404	"Баннер с данным id не найден"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/banner/{id} [delete]
//
//	@Security		AdminToken
func (bh *BannerHandlers) DeleteBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	// Получение уникального идентификатора
	id, err := strconv.ParseUint(c.Param(BannerIDField), 10, 64)
	if err != nil {
		tools.SendError(c, errors.Wrapf(err, "try get banner id"), http.StatusBadRequest, l)

		return
	}

	if err := bh.usecase.DeleteBanner(types.ID(id)); err != nil {
		if errors.Is(err, br.ErrorBannerNotFound) {
			tools.SendErrorStatus(c, err, http.StatusNotFound, l)

			return
		}

		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't delete banner"))

		return
	}

	tools.SendStatus(c, http.StatusNoContent, nil, l)
}

// UpdateBanner
//
//	@Summary		Обновление баннера.
//	@Description	Обновляет информацию о баннере по его id.
//	@Tags			banner
//	@Param			id	path	integer	true	"Идентификатор баннера"
//	@Accept			json
//	@Param			request	body	request.UpdateBanner	true	"Информация об обновлении"
//	@Produce		json
//	@Success		200	"Баннер успешно обновлён"
//	@Failure		400	{object}	tools.Error	"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		404	"Баннер с данным id не найден"
//	@Failure		409	"Баннер с указанной парой id фичи и ia тэга уже существует"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/banner/{id} [patch]
//
//	@Security		AdminToken
func (bh *BannerHandlers) UpdateBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	// Получение уникального идентификатора
	id, err := strconv.ParseUint(c.Param(BannerIDField), 10, 64)
	if err != nil {
		tools.SendError(c, errors.Wrapf(err, "try get banner id"), http.StatusBadRequest, l)

		return
	}

	// Получение значения тела запроса
	var updateBanner request.UpdateBanner
	if code, err := tools.ParseRequestBody(c.Request.Body, &updateBanner, request.ValidateUpdateBanner, l); err != nil {
		tools.SendError(c, err, code, l)

		return
	}

	if err := bh.usecase.UpdateBanner(types.ID(id), updateBanner.ToModel()); err != nil {
		if errors.Is(err, br.ErrorBannerNotFound) {
			tools.SendErrorStatus(c, err, http.StatusNotFound, l)

			return
		}

		if errors.Is(err, br.ErrorBannerConflictExists) {
			tools.SendErrorStatus(c, err, http.StatusConflict, l)

			return
		}

		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't update banner"))

		return
	}

	tools.SendStatus(c, http.StatusOK, nil, l)
}

// GetUserBanner
//
//	@Summary		Получение баннера для пользователя.
//	@Description	|
//					Возвращает баннер на основании тэга группы пользователей, фичи и версии, если версия не указана,
//					то вернётся последняя.
//
//	@Tags			banner
//	@Param			tag_id				query	integer	true	"Идентификатор тэга группы пользователей"
//	@Param			feature_id			query	integer	true	"Идентификатор фичи"
//	@Param			version				query	integer	false	"Версия баннера"
//	@Param			use_last_revision	query	boolean	false	"Получать актуальную информацию"
//	@Produce		json
//	@Success		200	{object}	any			"JSON-отображение баннера"
//	@Failure		400	{object}	tools.Error	"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		404	"Баннер с указанными тэгом и фичёй не найден"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/user_banner [get]
//
//	@Security		UserToken
func (bh *BannerHandlers) GetUserBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	tagID, err := tools.ParseQueryParamToTypesID(c, TagIDParam,
		ErrorTagIDNotPresented, ErrorTagIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	featureID, err := tools.ParseQueryParamToTypesID(c, FeatureIDParam,
		ErrorFeatureIDNotPresented, ErrorFeatureIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	version, err := tools.ParseQueryParamToUint32(c, VersionParam, nil, ErrorVersionIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	content, err := bh.usecase.GetUserBanner(*featureID, *tagID, version)
	if err != nil {
		if errors.Is(err, br.ErrorBannerNotFound) {
			tools.SendErrorStatus(c, err, http.StatusNotFound, l)

			return
		}

		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't get banner for user"))

		return
	}

	tools.SendStatus(c, http.StatusOK, content, l)

	if err := bh.cache.SetCache(*featureID, *tagID, version, types.Content(content)); err != nil {
		l.Error(errors.Wrapf(err,
			"can't cache banner with feature id %d, tag id %d and version %d", featureID, tagID, version))

		return
	}

	l.Info("banner with feature id %d, tag id %d and version %d was cached", featureID, tagID, version)
}

// GetAdminBanner
//
//	@Summary		Получение всех баннеров c фильтрацией по фиче и/или тегу
//	@Description	Возвращает список баннеров на основе фильтра по фиче и/или тегу.
//	@Tags			banner
//	@Param			tag_id		query	integer	false	"Идентификатор тэга группы пользователей"
//	@Param			feature_id	query	integer	false	"Идентификатор фичи"
//	@Param			limit		query	integer	false	"Лимит"
//	@Param			offset		query	integer	false	"Оффсет"
//	@Produce		json
//	@Success		200	{array}		response.Banner	"Список баннеров успешно отфильтрован"
//	@Failure		400	{object}	tools.Error		"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/banner [get]
//
//	@Security		AdminToken
func (bh *BannerHandlers) GetAdminBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	tagID, err := tools.ParseQueryParamToTypesID(c, TagIDParam, nil, ErrorTagIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	featureID, err := tools.ParseQueryParamToTypesID(c, FeatureIDParam, nil, ErrorFeatureIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	limit, err := tools.ParseQueryParamToUint64(c, limitParam, nil, ErrorLimitIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	offset, err := tools.ParseQueryParamToUint64(c, offsetParam, nil, ErrorOffsetIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	banners, err := bh.usecase.GetAdminBanners(featureID, tagID, offset, limit)
	if err != nil {
		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't get banners for admin"))

		return
	}

	tools.SendStatus(c, http.StatusOK, slices.Map(banners, func(banner *models.Banner) response.Banner {
		return *response.FromModelBanner(banner)
	}), l)
}

// DeleteFilterBanner
//
//	@Summary		Удаление всех баннеров c фильтрацией по фиче или тегу
//	@Description	Удаляет баннеры на основе фильтра по фиче или тегу. Обязателен один из query параметров.
//	@Tags			banner
//	@Param			tag_id		query	integer	false	"Идентификатор тэга группы пользователей"
//	@Param			feature_id	query	integer	false	"Идентификатор фичи"
//	@Produce		json
//	@Success		204	"Баннеры успешно удалены"
//	@Failure		400	{object}	tools.Error	"Некорректные данные"
//	@Failure		401	"Пользователь не авторизован"
//	@Failure		403	"Пользователь не имеет доступа"
//	@Failure		404	"Баннер с указанными тэгом и фичёй не найден"
//	@Failure		500	{object}	tools.Error	"Внутренняя ошибка сервера"
//	@Router			/filter_banner [delete]
//
//	@Security		AdminToken
func (bh *BannerHandlers) DeleteFilterBanner(c *gin.Context) {
	l := middleware.GetLogger(c)

	tagID, err := tools.ParseQueryParamToTypesID(c, TagIDParam, nil, ErrorTagIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	featureID, err := tools.ParseQueryParamToTypesID(c, FeatureIDParam, nil, ErrorFeatureIDIncorrectType, l)
	if err != nil {
		tools.SendError(c, err, http.StatusBadRequest, l)

		return
	}

	if featureID == nil && tagID == nil {
		tools.SendError(c, ErrorParamsNotPresented, http.StatusBadRequest, l)

		return
	}

	if err := bh.usecase.DeleteFilteredBanner(featureID, tagID); err != nil {
		if errors.Is(err, br.ErrorBannerNotFound) {
			tools.SendErrorStatus(c, err, http.StatusNotFound, l)

			return
		}

		tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
		l.Error(errors.Wrapf(err, "can't delete filtered banners"))

		return
	}

	tools.SendStatus(c, http.StatusNoContent, nil, l)
}
