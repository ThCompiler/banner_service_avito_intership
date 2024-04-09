package handlers

import (
	"bannersrv/external/auth"
	"bannersrv/internal/app/delivery/http/middleware"
	"bannersrv/internal/app/delivery/http/tools"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandlers struct {
	usecase auth.Usecase
}

func NewAuthHandlers(usecase auth.Usecase) *AuthHandlers {
	return &AuthHandlers{usecase: usecase}
}

// GetAdminToken
//
//	@Summary		Получение токена админа.
//	@Description	Возвращает токен с правами админа.
//	@Tags			auth
//	@Produce		json
//	@Success		200 {string} string	"Токен успешно создан"
//	@Router			/token/admin [get]
func (ah *AuthHandlers) GetAdminToken(c *gin.Context) {
	l := middleware.GetLogger(c)

	tools.SendStatus(c, http.StatusOK, ah.usecase.GetAdminToken(), l)
}

// GetUserToken
//
//	@Summary		Получение токена пользователя.
//	@Description	Возвращает токен с правами пользователя.
//	@Tags			auth
//	@Produce		json
//	@Success		200 {string} string	"Токен успешно создан"
//	@Router			/token/user [get]
func (ah *AuthHandlers) GetUserToken(c *gin.Context) {
	l := middleware.GetLogger(c)

	tools.SendStatus(c, http.StatusOK, ah.usecase.GetUserToken(), l)
}
