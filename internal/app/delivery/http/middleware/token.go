package middleware

import (
	"bannersrv/internal/app/delivery/http/tools"
	"bannersrv/internal/pkg/types"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	TokenHeaderField string = "token"

	TokenField types.ContextField = "token"
)

func RequestToken(c *gin.Context) {
	l := GetLogger(c)

	token := c.GetHeader(TokenHeaderField)

	if token == "" {
		l.Warn("token doesn't found in header of request")
		tools.SendStatus(c, http.StatusUnauthorized, nil, l)
		return
	}

	c.Set(string(TokenField), token)

	l.Info("handle request with token %s", token)
	c.Next()
}

func GetToken(c *gin.Context) string {
	if token, ok := c.Get(string(TokenField)); ok {
		return token.(string)
	}

	return ""
}
