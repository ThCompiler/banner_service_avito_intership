package middleware

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/caches"
	cr "bannersrv/internal/caches/repository"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

const (
	UseLastRevisionParam = "use_last_revision"
)

func CacheBanner(cacheManager caches.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := middleware.GetLogger(c)

		var useLastRevision = false
		if rawUseLastRevision, ok := c.GetQuery(UseLastRevisionParam); ok {
			tmp, err := strconv.ParseBool(rawUseLastRevision)
			if err != nil {
				l.Warn(errors.Wrapf(err, "can't parse query field %s with value %s",
					UseLastRevisionParam, rawUseLastRevision))
				tools.SendError(c, ErrorUseLastRevisionIncorrectType, http.StatusBadRequest, l)
				return
			}

			useLastRevision = tmp
		}

		if !useLastRevision {
			err := loadCache(c, cacheManager, l)
			if err == nil {
				return
			}

			if !errors.Is(err, cr.ErrorCacheMiss) {
				tools.SendError(c, err, http.StatusBadRequest, l)
				return
			}
		}

		c.Next()
	}
}

func loadCache(c *gin.Context, cacheManager caches.Manager, l logger.Interface) error {
	var tagId, featureId types.Id
	if rawTagId, err := tools.ParseQueryParamToUint64(c, handlers.TagIdParam,
		handlers.ErrorFeatureIdNotPresented, handlers.ErrorTagIdIncorrectType, l); err == nil {
		tagId = (types.Id)(rawTagId)
	} else {
		return err
	}

	if rawFeatureId, err := tools.ParseQueryParamToUint64(c, handlers.FeatureIdParam,
		handlers.ErrorFeatureIdNotPresented, handlers.ErrorFeatureIdIncorrectType, l); err == nil {
		featureId = (types.Id)(rawFeatureId)
	} else {
		return err
	}

	content, err := cacheManager.HaveCache(featureId, tagId)
	if err != nil {
		if !errors.Is(err, cr.ErrorCacheMiss) {
			l.Error(errors.Wrapf(err,
				"failed to check cached banner with feature id %d and tag id %d", featureId, tagId))
		}
		return cr.ErrorCacheMiss
	}

	tools.SendStatus(c, http.StatusOK, json.RawMessage(content), l)
	l.Info("banner wad loaded from cache with feature id %d and tag id %d", featureId, tagId)

	return nil
}
