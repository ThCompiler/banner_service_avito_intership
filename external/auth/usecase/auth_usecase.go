package usecase

import (
	"bannersrv/external/auth"
	"strings"

	"github.com/google/uuid"
)

const (
	adminPrefix = "admin"
	userPrefix  = "user"
)

type AuthUsecase struct{}

func NewAuthUsecase() *AuthUsecase {
	return &AuthUsecase{}
}

func (*AuthUsecase) IsAdminToken(token auth.Token) (bool, error) {
	if strings.HasPrefix(string(token), adminPrefix) {
		return true, nil
	}

	return false, nil
}

func (*AuthUsecase) IsUserToken(token auth.Token) (bool, error) {
	if strings.HasPrefix(string(token), userPrefix) {
		return true, nil
	}

	return false, nil
}

func (*AuthUsecase) GetUserToken() auth.Token {
	return auth.Token(userPrefix + "-" + uuid.New().String())
}

func (*AuthUsecase) GetAdminToken() auth.Token {
	return auth.Token(adminPrefix + "-" + uuid.New().String())
}
