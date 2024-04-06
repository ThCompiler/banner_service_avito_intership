package usecase

import (
	"bannersrv/external/auth"
	"github.com/google/uuid"
	"strings"
)

const (
	adminPrefix = "admin"
	userPrefix  = "user"
)

type AuthUsecase struct{}

func NewAuthUsecase() *AuthUsecase {
	return &AuthUsecase{}
}

func (au *AuthUsecase) IsAdminToken(token auth.Token) (bool, error) {
	if strings.HasPrefix(string(token), adminPrefix) {
		return true, nil
	}
	return false, nil
}

func (au *AuthUsecase) IsUserToken(token auth.Token) (bool, error) {
	if strings.HasPrefix(string(token), userPrefix) {
		return true, nil
	}
	return false, nil
}

func (au *AuthUsecase) GetUserToken() auth.Token {
	return auth.Token(userPrefix + "-" + uuid.New().String())
}

func (au *AuthUsecase) GetAdminToken() auth.Token {
	return auth.Token(adminPrefix + "-" + uuid.New().String())
}
