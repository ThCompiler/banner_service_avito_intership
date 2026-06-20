package usecase

import (
	"bannersrv/external/auth"
	"context"
	"strings"

	"github.com/ThCompiler/sdi"
	"github.com/google/uuid"
)

const (
	adminPrefix = "admin"
	userPrefix  = "user"
)

type AuthUsecase struct{}

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

func NewProvider() sdi.Provider[*AuthUsecase, struct{}] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, _ struct{}) (*AuthUsecase, error) {
		return &AuthUsecase{}, nil
	})
}
