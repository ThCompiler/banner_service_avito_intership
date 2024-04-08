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

		var skipCache = false
		if rawUseLastRevision, ok := c.GetQuery(UseLastRevisionParam); ok {
			tmp, err := strconv.ParseBool(rawUseLastRevision)
			if err != nil {
				l.Warn(errors.Wrapf(err, "can't parse query field %s with value %s",
					UseLastRevisionParam, rawUseLastRevision))
				tools.SendError(c, ErrorUseLastRevisionIncorrectType, http.StatusBadRequest, l)
				return
			}

			skipCache = tmp
		}

		if !skipCache {
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
	var version = new(uint32)

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

	if rawVersion, err := tools.ParseQueryParamToUint64(c, handlers.VersionParam,
		handlers.ErrorVersionNotPresented, handlers.ErrorVersionIncorrectType, l); err == nil {
		*version = uint32(rawVersion)
	} else {
		if !errors.Is(err, handlers.ErrorVersionNotPresented) {
			return err
		}
		version = nil
	}

	content, err := cacheManager.HaveCache(featureId, tagId, version)
	if err != nil {
		if !errors.Is(err, cr.ErrorCacheMiss) {
			l.Error(errors.Wrapf(err,
				"failed to check cached banner with feature id %d, tag id %d, version %d", featureId, tagId, version))
		}
		return cr.ErrorCacheMiss
	}

	tools.SendStatus(c, http.StatusOK, json.RawMessage(content), l)
	l.Info("banner wad loaded from cache with feature id %d and tag id %d, version %d", featureId, tagId, version)

	return nil
}
