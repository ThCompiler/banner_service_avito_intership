package tools

import (
	"bannersrv/internal/pkg/evjson"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var (
	ErrorCannotReadBody       = errors.New("can't read body")
	ErrorIncorrectBodyContent = errors.New("incorrect body content")
)

const (
	size32 = 32
	size64 = 64
	base10 = 10
)

func ParseRequestBody(reqBody io.ReadCloser, out any, validation func([]byte) error, l logger.Interface) (int, error) {
	body, err := io.ReadAll(reqBody)
	if err != nil {
		l.Error(errors.Wrapf(err, "can't read body"))

		return http.StatusInternalServerError, ErrorCannotReadBody
	}

	// Проверка корректности тела запроса
	if err := validation(body); err != nil {
		if errors.Is(err, evjson.ErrorInvalidJSON) {
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

// ParseQueryParamToTypesID преобразует параметр запроса в types.ID
// Если ошибка notPresentedError установлена в nil, то будет возвращаться nil в качестве ошибки и в качестве значения
func ParseQueryParamToTypesID(c *gin.Context, param string, notPresentedError error,
	incorrectTypeError error, l logger.Interface,
) (*types.ID, error) {
	if rawField, ok := c.GetQuery(param); ok {
		id, err := strconv.ParseUint(rawField, base10, size32)
		if err != nil {
			l.Error(errors.Wrapf(err, "can't parse query field %s with value %s", param, rawField))

			return nil, incorrectTypeError
		}

		resID := types.ID(id)

		return &resID, nil
	}

	return nil, notPresentedError
}

// ParseQueryParamToUint32 преобразует параметр запроса в uint32
// Если ошибка notPresentedError установлена в nil, то будет возвращаться nil в качестве ошибки и в качестве значения
func ParseQueryParamToUint32(c *gin.Context, param string, notPresentedError error,
	incorrectTypeError error, l logger.Interface,
) (*uint32, error) {
	if rawField, ok := c.GetQuery(param); ok {
		id, err := strconv.ParseUint(rawField, base10, size32)
		if err != nil {
			l.Error(errors.Wrapf(err, "can't parse query field %s with value %s", param, rawField))

			return nil, incorrectTypeError
		}

		resID := uint32(id)

		return &resID, nil
	}

	return nil, notPresentedError
}

// ParseQueryParamToUint64 преобразует параметр запроса в uint64
// Если ошибка notPresentedError установлена в nil, то будет возвращаться nil в качестве ошибки и в качестве значения
func ParseQueryParamToUint64(c *gin.Context, param string, notPresentedError error,
	incorrectTypeError error, l logger.Interface,
) (*uint64, error) {
	if rawField, ok := c.GetQuery(param); ok {
		id, err := strconv.ParseUint(rawField, base10, size64)
		if err != nil {
			l.Error(errors.Wrapf(err, "can't parse query field %s with value %s", param, rawField))

			return nil, incorrectTypeError
		}

		return &id, nil
	}

	return nil, notPresentedError
}
