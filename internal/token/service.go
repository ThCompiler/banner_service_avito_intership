package token

import (
	"bannersrv/external/auth"
)

type Service interface {
	IsAdminToken(token auth.Token) (bool, error)
	IsUserToken(token auth.Token) (bool, error)
}
