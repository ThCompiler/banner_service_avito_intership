package auth

type Token string

type Usecase interface {
	GetUserToken() Token
	GetAdminToken() Token
}
