package v1

import (
	"bannersrv/internal/app/delivery/http/middleware"
	"net/http"

	"github.com/gin-gonic/gin"

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

func NewRouter(root string, l logger.Interface, routes Routes) (*gin.Engine, error) {
	router := gin.New()

	router.Use(middleware.RequestToken, middleware.CheckPanic, middleware.RequestLogger(l))
	rt := router.Group(root)
	v1 := rt.Group(version)

	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			v1.GET(route.Pattern, route.HandlerFunc).Use(route.Middlewares...)
		case http.MethodPost:
			v1.POST(route.Pattern, route.HandlerFunc).Use(route.Middlewares...)
		case http.MethodPut:
			v1.PUT(route.Pattern, route.HandlerFunc).Use(route.Middlewares...)
		case http.MethodDelete:
			v1.DELETE(route.Pattern, route.HandlerFunc).Use(route.Middlewares...)
		case http.MethodOptions:
			v1.OPTIONS(route.Pattern, route.HandlerFunc).Use(route.Middlewares...)
		}
	}

	return router, nil
}
