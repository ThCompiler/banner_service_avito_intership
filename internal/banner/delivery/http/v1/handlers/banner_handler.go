package handlers

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/banner"
	"bannersrv/internal/banner/delivery/http/v1/models/request"
	"bannersrv/internal/banner/delivery/http/v1/models/response"
	"bannersrv/internal/banner/models"
	br "bannersrv/internal/banner/repository"
	"bannersrv/internal/caches"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/slices"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

const BannerIdField = "id"

const (
	TagIdParam     = "tag_id"
	FeatureIdParam = "feature_id"
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
//	@Success		201	{object}	response.BannerId	"Баннер успешно добавлен в систему"
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

	createdId, err := bh.usecase.CreateBanner(createBanner.TagsIds, createBanner.FeatureId,
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

	tools.SendStatus(c, http.StatusCreated, &response.BannerId{BannerId: createdId}, l)
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
	id, err := strconv.ParseUint(c.Param(BannerIdField), 10, 64)
	if err != nil {
		tools.SendError(c, errors.Wrapf(err, "try get banner id"), http.StatusBadRequest, l)
		return
	}

	if err := bh.usecase.DeleteBanner(types.Id(id)); err != nil {
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
	id, err := strconv.ParseUint(c.Param(BannerIdField), 10, 64)
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

	if err := bh.usecase.UpdateBanner(types.Id(id), updateBanner.ToModel()); err != nil {
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
//	@Description	Возвращает баннер на основании тэга группы пользователей, фичи и версии, если версия не указана, то вернётся последняя.
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

	var tagId, featureId types.Id
	var version = new(uint32)

	if rawTagId, err := tools.ParseQueryParamToUint64(c, TagIdParam,
		ErrorTagIdNotPresented, ErrorTagIdIncorrectType, l); err == nil {
		tagId = types.Id(rawTagId)
	} else {
		tools.SendError(c, err, http.StatusBadRequest, l)
		return
	}

	if rawFeatureId, err := tools.ParseQueryParamToUint64(c, FeatureIdParam,
		ErrorFeatureIdNotPresented, ErrorFeatureIdIncorrectType, l); err == nil {
		featureId = types.Id(rawFeatureId)
	} else {
		tools.SendError(c, err, http.StatusBadRequest, l)
		return
	}

	if rawVersion, err := tools.ParseQueryParamToUint64(c, VersionParam,
		ErrorVersionNotPresented, ErrorVersionIncorrectType, l); err == nil {
		*version = uint32(rawVersion)
	} else {
		if !errors.Is(err, ErrorVersionNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		version = nil
	}

	content, err := bh.usecase.GetUserBanner(featureId, tagId, version)
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

	if err := bh.cache.SetCache(featureId, tagId, version, types.Content(content)); err != nil {
		l.Error(errors.Wrapf(err,
			"can't cache banner with feature id %d, tag id %d and version %d", featureId, tagId, version))
		return
	}
	l.Info("banner with feature id %d, tag id %d and version %d was cached", featureId, tagId, version)
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

	var tagId, featureId = new(types.Id), new(types.Id)
	var offset, limit = new(uint64), new(uint64)

	if rawTagId, err := tools.ParseQueryParamToUint64(c, TagIdParam,
		ErrorTagIdNotPresented, ErrorTagIdIncorrectType, l); err == nil {
		*tagId = (types.Id)(rawTagId)
	} else {
		if !errors.Is(err, ErrorTagIdNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		tagId = nil
	}

	if rawFeatureId, err := tools.ParseQueryParamToUint64(c, FeatureIdParam,
		ErrorFeatureIdNotPresented, ErrorFeatureIdIncorrectType, l); err == nil {
		*featureId = (types.Id)(rawFeatureId)
	} else {
		if !errors.Is(err, ErrorFeatureIdNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		featureId = nil
	}

	if rawLimit, err := tools.ParseQueryParamToUint64(c, limitParam,
		ErrorLimitNotPresented, ErrorLimitIncorrectType, l); err == nil {
		*limit = rawLimit
	} else {
		if !errors.Is(err, ErrorLimitNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		limit = nil
	}

	if rawOffset, err := tools.ParseQueryParamToUint64(c, offsetParam,
		ErrorOffsetNotPresented, ErrorOffsetIncorrectType, l); err == nil {
		*offset = rawOffset
	} else {
		if !errors.Is(err, ErrorOffsetNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		offset = nil
	}

	banners, err := bh.usecase.GetAdminBanners(featureId, tagId, offset, limit)
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

	var tagId, featureId = new(types.Id), new(types.Id)

	if rawTagId, err := tools.ParseQueryParamToUint64(c, TagIdParam,
		ErrorTagIdNotPresented, ErrorTagIdIncorrectType, l); err == nil {
		*tagId = (types.Id)(rawTagId)
	} else {
		if !errors.Is(err, ErrorTagIdNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		tagId = nil
	}

	if rawFeatureId, err := tools.ParseQueryParamToUint64(c, FeatureIdParam,
		ErrorFeatureIdNotPresented, ErrorFeatureIdIncorrectType, l); err == nil {
		*featureId = (types.Id)(rawFeatureId)
	} else {
		if !errors.Is(err, ErrorFeatureIdNotPresented) {
			tools.SendError(c, err, http.StatusBadRequest, l)
			return
		}
		featureId = nil
	}

	if featureId == nil && tagId == nil {
		tools.SendError(c, ErrorParamsNotPresented, http.StatusBadRequest, l)
		return
	}

	err := bh.usecase.DeleteFilteredBanner(featureId, tagId)
	if err != nil {
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
