package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager 负责 JWT 的签发与解析。
type TokenManager struct {
	secret       []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	mfaTicketTTL time.Duration // 2FA 中间票据有效期
}

// NewTokenManager 构造令牌管理器。
// mfaTicketTTL 为 2FA 中间票据有效期，传 0 时使用默认值 5 分钟。
func NewTokenManager(secret string, accessTTL, refreshTTL, mfaTicketTTL time.Duration) *TokenManager {
	if mfaTicketTTL <= 0 {
		mfaTicketTTL = DefaultMFATicketTTL
	}
	return &TokenManager{
		secret:       []byte(secret),
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		mfaTicketTTL: mfaTicketTTL,
	}
}

// AccessTTL 返回 access token 有效期。
func (m *TokenManager) AccessTTL() time.Duration { return m.accessTTL }

// RefreshTTL 返回 refresh token 有效期。
func (m *TokenManager) RefreshTTL() time.Duration { return m.refreshTTL }

// GenerateAccess 签发 access token，jti 用于后续黑名单标识。
func (m *TokenManager) GenerateAccess(userUUID, jti string) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		Type: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// GenerateRefresh 签发 refresh token，绑定设备 ID 与设备指纹。
func (m *TokenManager) GenerateRefresh(userUUID, tokenID, deviceID, fingerprint string) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		Type:              TokenTypeRefresh,
		TokenID:           tokenID,
		DeviceID:          deviceID,
		DeviceFingerprint: fingerprint,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID,
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// GenerateMFATicket 签发 2FA 中间票据。
//
// 用途：账号密码校验通过后，若用户已开启 2FA，则先不下发令牌，
// 而是签发一张短时效（默认 5 分钟）的票据；用户提交验证码后凭此票据
// 换取真正的双令牌。票据只用于关联登录的两个步骤，不代表已登录。
func (m *TokenManager) GenerateMFATicket(userUUID, jti string) (string, error) {
	now := time.Now()
	claims := MFATicketClaims{
		Type: TokenTypeMFATicket,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.mfaTicketTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ParseMFATicket 解析并校验 2FA 中间票据（验签 + 过期 + 类型）。
func (m *TokenManager) ParseMFATicket(tokenStr string) (*MFATicketClaims, error) {
	claims := &MFATicketClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, m.keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("2FA 票据无效: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("2FA 票据无效")
	}
	if claims.Type != TokenTypeMFATicket {
		return nil, fmt.Errorf("令牌类型不匹配")
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("2FA 票据缺少 sub 声明")
	}
	return claims, nil
}

// ParseAccess 解析并校验 access token（验签 + 过期 + 类型）。
func (m *TokenManager) ParseAccess(tokenStr string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, m.keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("访问令牌无效: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("访问令牌无效")
	}
	if claims.Type != TokenTypeAccess {
		return nil, fmt.Errorf("令牌类型不匹配")
	}
	return claims, nil
}

// ParseRefresh 解析并校验 refresh token（验签 + 过期 + 类型）。
// 白名单与设备指纹校验由 AuthService 完成。
func (m *TokenManager) ParseRefresh(tokenStr string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, m.keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("刷新令牌无效: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("刷新令牌无效")
	}
	if claims.Type != TokenTypeRefresh {
		return nil, fmt.Errorf("令牌类型不匹配")
	}
	if claims.TokenID == "" || claims.Subject == "" {
		return nil, fmt.Errorf("缺少 tokenId 或 sub 声明")
	}
	return claims, nil
}

func (m *TokenManager) keyFunc(t *jwt.Token) (any, error) {
	return m.secret, nil
}
