package tools

import (
	"bannersrv/internal/pkg/evjson"
	"bannersrv/pkg/logger"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strconv"
)

var (
	ErrorCannotReadBody       = errors.New("can't read body")
	ErrorIncorrectBodyContent = errors.New("incorrect body content")
)

func ParseRequestBody(reqBody io.ReadCloser, out any, validation func([]byte) error, l logger.Interface) (int, error) {
	body, err := io.ReadAll(reqBody)
	if err != nil {
		l.Error(errors.Wrapf(err, "can't read body"))
		return http.StatusInternalServerError, ErrorCannotReadBody
	}

	// Проверка корректности тела запроса
	if err := validation(body); err != nil {
		if errors.Is(err, evjson.ErrorInvalidJson) {
			l.Warn(errors.Wrapf(err, "try parse body json"))
			return http.StatusBadRequest, ErrorIncorrectBodyContent
		}
		return http.StatusBadRequest, errors.Wrapf(err, "in body error")
	}

	// Получение значения тела запроса
	if err := json.Unmarshal(body, out); err != nil {
		l.Warn(errors.Wrapf(err, "try parse create request entity"))
		return http.StatusBadRequest, ErrorIncorrectBodyContent
	}

	return http.StatusOK, nil
}

func ParseQueryParamToUint64(c *gin.Context, param string, notPresentedError error,
	incorrectTypeError error, l logger.Interface) (uint64, error) {
	if rawFeatureId, ok := c.GetQuery(param); ok {
		id, err := strconv.ParseUint(rawFeatureId, 10, 64)
		if err != nil {
			l.Error(errors.Wrapf(err, "can't parse query field %s with value %s", param, rawFeatureId))
			return 0, incorrectTypeError
		}
		return id, nil
	}

	return 0, notPresentedError
}
