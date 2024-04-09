package v1

import (
	"bannersrv/internal/app/config"
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/pkg/metrics"
	"bannersrv/pkg/logger"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "v1"

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
	Middlewares []gin.HandlerFunc
}

type Routes []Route

func NewRouter(root string, routes Routes, mode config.Mode,
	l logger.Interface, metricsManager metrics.Manager,
) (*gin.Engine, error) {
	if mode == config.Release || mode == config.ReleaseProf {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	if mode == config.DebugProf || mode == config.ReleaseProf {
		pprof.Register(router)
	}

	promHandler := promhttp.Handler()

	router.GET("/metrics", func(c *gin.Context) {
		promHandler.ServeHTTP(c.Writer, c.Request)
	})

	router.Use(middleware.RequestLogger(l), middleware.CheckPanic, middleware.RequestMetrics(metricsManager))
	rt := router.Group(root)
	v1 := rt.Group(version)

	for _, route := range routes {
		route.Middlewares = append(route.Middlewares, route.HandlerFunc)
		v1.Handle(route.Method, route.Pattern, route.Middlewares...)
	}

	return router, nil
}
