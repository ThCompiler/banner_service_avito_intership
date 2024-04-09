package middleware

import (
	"bannersrv/pkg/logger"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// CheckPanic обрабатывает панические ситуации, которые могли возникнуть при обработке запросов
func CheckPanic(c *gin.Context) {
	defer func(log logger.Interface, c *gin.Context) {
		if err := recover(); err != nil {
			log.Error("detected critical error: %v, with stack: %s", err, debug.Stack())
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}(GetLogger(c), c)

	// Process request
	c.Next()
}
