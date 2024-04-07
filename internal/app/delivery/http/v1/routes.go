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

	router.Use(middleware.RequestLogger(l), middleware.CheckPanic)
	rt := router.Group(root)
	v1 := rt.Group(version)

	for _, route := range routes {
		route.Middlewares = append(route.Middlewares, route.HandlerFunc)
		switch route.Method {
		case http.MethodGet:
			v1.GET(route.Pattern, route.Middlewares...)
		case http.MethodPost:
			v1.POST(route.Pattern, route.Middlewares...)
		case http.MethodHead:
			v1.HEAD(route.Pattern, route.Middlewares...)
		case http.MethodPut:
			v1.PUT(route.Pattern, route.Middlewares...)
		case http.MethodPatch:
			v1.PATCH(route.Pattern, route.Middlewares...)
		case http.MethodDelete:
			v1.DELETE(route.Pattern, route.Middlewares...)
		case http.MethodOptions:
			v1.OPTIONS(route.Pattern, route.Middlewares...)
		}
	}

	return router, nil
}
