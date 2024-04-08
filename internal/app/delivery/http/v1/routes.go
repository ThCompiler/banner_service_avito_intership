package v1

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"bannersrv/pkg/logger"
)

const version = "v1"

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
	Middlewares []gin.HandlerFunc
}

type Routes []Route

func NewRouter(root string, l logger.Interface, routes Routes, metricsManager metrics.Manager) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	router.Use(middleware.RequestLogger(l, metricsManager), middleware.CheckPanic)
	rt := router.Group(root)
	v1 := rt.Group(version)

	for _, route := range routes {
		route.Middlewares = append(route.Middlewares, route.HandlerFunc)
		v1.Handle(route.Method, route.Pattern, route.Middlewares...)
	}

	return router, nil
}
