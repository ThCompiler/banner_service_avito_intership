package app

import (
	"bannersrv/internal/app/config"
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/caches"
	"bannersrv/internal/pkg/prepare"
	"bannersrv/internal/token"
	"bannersrv/pkg/logger"
	"io"
	"log"
	"net/http"
	"os"

	ah "bannersrv/external/auth/delivery/http/v1/handlers"

	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"

	cm "bannersrv/internal/caches/delivery/middleware"

	tm "bannersrv/internal/token/delivery/middleware"

	"github.com/gin-gonic/gin"

	sf "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"

	_ "bannersrv/docs"
)

func prepareLogger(cfg config.LoggerInfo) (*logger.Logger, *os.File) {
	var logOut io.Writer

	var logFile *os.File

	var err error

	if cfg.Directory != "" {
		logFile, err = prepare.OpenLogDir(cfg.Directory)
		if err != nil {
			log.Fatalf("[App] Init - create logger error: %s", err) // nolint: revive // логгер инициализируется,
			// ошибку открытия лог файла больше нечем логировать
		}

		logOut = logFile
	} else {
		logOut = os.Stderr
		logFile = nil
	}

	l := logger.New(
		logger.Params{
			AppName:                  cfg.AppName,
			LogDir:                   cfg.Directory,
			Level:                    cfg.Level,
			UseStdAndFile:            cfg.UseStdAndFile,
			AddLowPriorityLevelToCmd: cfg.AllowShowLowLevel,
		},
		logOut,
	)

	return l, logFile
}

func PrepareRoutes(bannerHandlers *bh.BannerHandlers, cache caches.Manager,
	tokenService token.Service, authHandlers *ah.AuthHandlers,
) v1.Routes {
	return v1.Routes{
		// "Swagger"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/swagger/*any",
			HandlerFunc: gs.WrapHandler(sf.Handler),
		},

		// "CreateBanner"
		v1.Route{
			Method:      http.MethodPost,
			Pattern:     "/banner",
			HandlerFunc: bannerHandlers.CreateBanner,
			Middlewares: []gin.HandlerFunc{middleware.RequestToken, tm.WithAdminToken(tokenService)},
		},

		// "GetAdminBanner"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/banner",
			HandlerFunc: bannerHandlers.GetAdminBanner,
			Middlewares: []gin.HandlerFunc{middleware.RequestToken, tm.WithAdminToken(tokenService)},
		},

		// "DeleteBanner"
		v1.Route{
			Method:      http.MethodDelete,
			Pattern:     "/banner/:" + bh.BannerIDField,
			HandlerFunc: bannerHandlers.DeleteBanner,
			Middlewares: []gin.HandlerFunc{middleware.RequestToken, tm.WithAdminToken(tokenService)},
		},

		// "UpdateBanner"
		v1.Route{
			Method:      http.MethodPatch,
			Pattern:     "/banner/:" + bh.BannerIDField,
			HandlerFunc: bannerHandlers.UpdateBanner,
			Middlewares: []gin.HandlerFunc{middleware.RequestToken, tm.WithAdminToken(tokenService)},
		},

		// "GetUserBanner"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/user_banner",
			HandlerFunc: bannerHandlers.GetUserBanner,
			Middlewares: []gin.HandlerFunc{
				middleware.RequestToken,
				tm.WithUserToken(tokenService), cm.CacheBanner(cache),
			},
		},

		// "DeleteFilterBanner"
		v1.Route{
			Method:      http.MethodDelete,
			Pattern:     "/filter_banner",
			HandlerFunc: bannerHandlers.DeleteFilterBanner,
			Middlewares: []gin.HandlerFunc{middleware.RequestToken, tm.WithAdminToken(tokenService)},
		},

		// Для эмуляции сервиса выдачи токенов
		// "GetAdminToken"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/token/admin",
			HandlerFunc: authHandlers.GetAdminToken,
		},

		// "GetUserToken"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/token/user",
			HandlerFunc: authHandlers.GetUserToken,
		},
	}
}
