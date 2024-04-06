package app

import (
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	"bannersrv/internal/caches"
	cm "bannersrv/internal/caches/delivery/middleware"
	"bannersrv/internal/token"
	tm "bannersrv/internal/token/delivery/middleware"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"bannersrv/config"
	_ "bannersrv/docs"
	"bannersrv/internal/pkg/prepare"
	"bannersrv/pkg/logger"
)

func prepareLogger(cfg config.LoggerInfo) (*logger.Logger, *os.File) {
	var logOut io.Writer
	var logFile *os.File
	var err error

	if cfg.Directory != "" {
		logFile, err = prepare.OpenLogDir(cfg.Directory)
		if err != nil {
			log.Fatalf("[App] Init - create logger error: %s", err)
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

func prepareRoutes(bannerHandlers *bh.BannerHandlers, cache caches.Manager,
	tokenService token.Service, authHandlers *ah.AuthHandlers) v1.Routes {
	return v1.Routes{
		//"Index"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/swagger/*any",
			HandlerFunc: ginSwagger.WrapHandler(swaggerFiles.Handler),
		},

		// "CreateBanner"
		v1.Route{
			Method:      http.MethodPost,
			Pattern:     "/banner",
			HandlerFunc: bannerHandlers.CreateBanner,
			Middlewares: []gin.HandlerFunc{tm.WithAdminToken(tokenService)},
		},

		// "GetAdminBanner"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/banner",
			HandlerFunc: bannerHandlers.GetAdminBanner,
			Middlewares: []gin.HandlerFunc{tm.WithAdminToken(tokenService)},
		},

		// "DeleteBanner"
		v1.Route{
			Method:      http.MethodDelete,
			Pattern:     "/banner/:" + bh.BannerIdField,
			HandlerFunc: bannerHandlers.DeleteBanner,
			Middlewares: []gin.HandlerFunc{tm.WithAdminToken(tokenService)},
		},

		// "UpdateBanner"
		v1.Route{
			Method:      http.MethodPatch,
			Pattern:     "/banner/:" + bh.BannerIdField,
			HandlerFunc: bannerHandlers.UpdateBanner,
			Middlewares: []gin.HandlerFunc{tm.WithAdminToken(tokenService)},
		},

		// "GetUserBanner"
		v1.Route{
			Method:      http.MethodGet,
			Pattern:     "/user_banner",
			HandlerFunc: bannerHandlers.GetUserBanner,
			Middlewares: []gin.HandlerFunc{tm.WithUserToken(tokenService), cm.CacheBanner(cache)},
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
