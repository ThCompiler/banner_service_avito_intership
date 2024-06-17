package tools

import (
	"bannersrv/pkg/logger"

	"github.com/gin-gonic/gin"
)

func SendStatus(c *gin.Context, code int, data any, l logger.Interface) {
	l.Info("was sent response with status code %d", code)

	if data != nil {
		c.AbortWithStatusJSON(code, data)
	}

	c.AbortWithStatus(code)
}
