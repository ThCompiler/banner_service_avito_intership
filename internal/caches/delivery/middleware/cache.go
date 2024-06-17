package middleware

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/caches"
	"bannersrv/pkg/logger"
	"encoding/json"
	"net/http"
	"strconv"

	cr "bannersrv/internal/caches/repository"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	UseLastRevisionParam = "use_last_revision"
)

func CacheBanner(cacheManager caches.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := middleware.GetLogger(c)

		skipCache := false

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
	tagID, err := tools.ParseQueryParamToTypesID(c, handlers.TagIDParam,
		handlers.ErrorFeatureIDNotPresented, handlers.ErrorTagIDIncorrectType, l)
	if err != nil {
		return err
	}

	featureID, err := tools.ParseQueryParamToTypesID(c, handlers.FeatureIDParam,
		handlers.ErrorFeatureIDNotPresented, handlers.ErrorFeatureIDIncorrectType, l)
	if err != nil {
		return err
	}

	version, err := tools.ParseQueryParamToUint32(c, handlers.VersionParam,
		nil, handlers.ErrorVersionIncorrectType, l)
	if err != nil {
		return err
	}

	content, err := cacheManager.HaveCache(*featureID, *tagID, version)
	if err != nil {
		if !errors.Is(err, cr.ErrorCacheMiss) {
			l.Error(errors.Wrapf(err,
				"failed to check cached banner with feature id %d, tag id %d, version %d", featureID, tagID, version))
		}

		return cr.ErrorCacheMiss
	}

	tools.SendStatus(c, http.StatusOK, json.RawMessage(content), l)
	l.Info("banner wad loaded from cache with feature id %d and tag id %d, version %d", featureID, tagID, version)

	return nil
}
