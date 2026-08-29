// Package mfa 实现基于 TOTP（RFC 6238）的双因素认证。
//
// 职责边界：本包只做「密钥与验证码的算法层」工作——生成密钥、构造
// otpauth:// 绑定 URL、校验验证码、生成与校验恢复码；
// 不涉及数据库读写与令牌签发，这些由 auth 包编排。
//
// 依赖方向：mfa 仅依赖标准库与 pquerna/otp，不依赖任何内部包，
// 因此可被 auth / handler 安全引用，不会形成循环依赖。
package mfa

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// 默认参数，符合 Google Authenticator 等主流 App 的兼容要求。
const (
	// DefaultIssuer 默认发行方名称，展示在验证器 App 中。
	DefaultIssuer = "MuseFlow"
	// DefaultDigits 验证码位数，主流 App 均为 6 位。
	DefaultDigits = 6
	// DefaultPeriod 时间步长（秒），TOTP 标准默认 30 秒。
	DefaultPeriod = 30
	// DefaultRecoveryCodeCount 恢复码数量。
	DefaultRecoveryCodeCount = 8
	// DefaultRecoveryCodeLength 单个恢复码的字符长度（不含分隔符）。
	DefaultRecoveryCodeLength = 10
)

// Secret 表示一个 TOTP 共享密钥及其绑定信息。
type Secret struct {
	// Raw 原始密钥（base32 编码，无填充），需存入数据库。
	Raw string
	// URL 符合 otpauth:// 协议的绑定地址，供前端生成二维码。
	URL string
}

// GenerateSecret 生成新的 TOTP 共享密钥及绑定 URL。
//
// accountName 通常使用用户邮箱，会展示在验证器 App 中；
// issuer 为发行方名称，传空则使用 DefaultIssuer。
func GenerateSecret(accountName, issuer string) (*Secret, error) {
	if accountName == "" {
		return nil, fmt.Errorf("账号名不能为空")
	}
	if issuer == "" {
		issuer = DefaultIssuer
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Digits:      otp.DigitsSix,
		Period:      DefaultPeriod,
	})
	if err != nil {
		return nil, fmt.Errorf("生成 TOTP 密钥失败: %w", err)
	}

	return &Secret{Raw: key.Secret(), URL: key.URL()}, nil
}

// BuildURL 根据已有密钥重新构造绑定 URL（用于重新展示二维码）。
func BuildURL(secret, accountName, issuer string) string {
	if issuer == "" {
		issuer = DefaultIssuer
	}
	label := url.QueryEscape(issuer) + ":" + url.QueryEscape(accountName)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", DefaultDigits))
	q.Set("period", fmt.Sprintf("%d", DefaultPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// ValidateCode 校验 TOTP 验证码是否合法。
//
// skew 为允许的时间偏移步数，用于容忍客户端与服务器的时钟误差；
// 传 0 时使用默认值 1（即允许前后各 1 个时间窗口，共约 ±30 秒）。
func ValidateCode(code, secret string, skew int) bool {
	if code == "" || secret == "" {
		return false
	}
	if skew <= 0 {
		skew = 1
	}
	clean := strings.TrimSpace(code)
	ok, err := totp.ValidateCustom(clean, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    DefaultPeriod,
		Skew:      uint(skew),
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return ok
}

// GenerateRecoveryCodes 生成指定数量的恢复码。
//
// 返回明文列表，调用方需自行哈希后存储；明文只在生成这一次返回给用户。
func GenerateRecoveryCodes(count, length int) ([]string, error) {
	if count <= 0 {
		count = DefaultRecoveryCodeCount
	}
	if length <= 0 {
		length = DefaultRecoveryCodeLength
	}

	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(codes) < count {
		code, err := randomRecoveryCode(length)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes, nil
}

// randomRecoveryCode 生成形如 "a1b2c-3d4e5" 的恢复码（含一个分隔符便于抄写）。
func randomRecoveryCode(length int) (string, error) {
	// 去掉易混淆字符 0/O/1/l/I，降低抄写错误率
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	half := length / 2

	var sb strings.Builder
	for i := 0; i < length; i++ {
		if i == half {
			sb.WriteByte('-')
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("生成恢复码失败: %w", err)
		}
		sb.WriteByte(alphabet[n.Int64()])
	}
	return sb.String(), nil
}

// MatchRecoveryCode 在已存储的恢复码中常量时间比对给定恢复码。
//
// stored 为数据库保存的恢复码列表（建议存哈希，此处按明文比对并调用方保证），
// 返回匹配的下标与是否命中；未命中返回 -1。
func MatchRecoveryCode(input string, stored []string) int {
	clean := normalizeRecoveryCode(input)
	if clean == "" {
		return -1
	}
	for i, s := range stored {
		if constantTimeEqual(clean, normalizeRecoveryCode(s)) {
			return i
		}
	}
	return -1
}

// normalizeRecoveryCode 统一恢复码格式：去空格、转小写、移除分隔符。
func normalizeRecoveryCode(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// constantTimeEqual 常量时间字符串比较，降低时序侧信道风险。
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// IsValidSecretFormat 检查密钥是否为合法的 base32 字符串（用于入库前校验）。
func IsValidSecretFormat(secret string) bool {
	if secret == "" {
		return false
	}
	// 补齐 base32 所需的 '=' 填充后尝试解码
	pad := (8 - len(secret)%8) % 8
	if _, err := base32.StdEncoding.DecodeString(secret + strings.Repeat("=", pad)); err != nil {
		return false
	}
	return true
}
