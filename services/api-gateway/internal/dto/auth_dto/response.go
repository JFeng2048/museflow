package authdto

import (
	"github.com/museflow/api-gateway/internal/dto/user_dto"
)

// LoginData 登录成功返回的数据。
// refresh token 不在 body 中返回，而是写入 HttpOnly Cookie。
type LoginData struct {
	AccessToken string           `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string           `json:"token_type" example:"Bearer"`
	ExpiresIn   int64            `json:"expires_in" example:"3600"`
	User        userdto.UserInfo `json:"user"`
}

// RefreshData 刷新成功返回的数据。
type RefreshData struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"3600"`
}
