package tools

import (
	"bannersrv/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var ErrorServerError = errors.New("some server error, try again later")

type Error struct {
	Error string `json:"error,omitempty"`
}

func SendError(c *gin.Context, err error, code int, l logger.Interface) {
	c.AbortWithStatusJSON(code, Error{Error: err.Error()})
	l.Info("error %s was sent with status code %d", err, code)
}

func SendErrorStatus(c *gin.Context, err error, code int, l logger.Interface) {
	c.AbortWithStatus(code)
	l.Info("error %s was sent with status code %d", err, code)
}
