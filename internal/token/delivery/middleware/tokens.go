package middleware

import (
	"bannersrv/external/auth"
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

func WithAdminToken(tokenService token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := middleware.GetLogger(c)

		tok := middleware.GetToken(c)

		ok, err := tokenService.IsAdminToken(auth.Token(tok))
		if err != nil {
			tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
			l.Error(errors.Wrapf(err, "try check admin token %s", tok))
			return
		}

		if !ok {
			tools.SendStatus(c, http.StatusForbidden, nil, l)
			return
		}

		l.Info("handle request with admin permissions")
		c.Next()
	}
}

func WithUserToken(tokenService token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := middleware.GetLogger(c)

		tok := middleware.GetToken(c)

		ok, err := tokenService.IsUserToken(auth.Token(tok))
		if err != nil {
			tools.SendError(c, tools.ErrorServerError, http.StatusInternalServerError, l)
			l.Error(errors.Wrapf(err, "try check user token %s", tok))
			return
		}

		if !ok {
			tools.SendStatus(c, http.StatusForbidden, nil, l)
			return
		}

		l.Info("handle request with user permissions")
		c.Next()
	}
}
