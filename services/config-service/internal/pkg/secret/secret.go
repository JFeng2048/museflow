// Package secret 提供模型凭证的对称加密。
//
// config_svc 下的 model_provider.api_key / user_model_provider.api_key 与
// system_setting.secret_value 都不允许明文落库，统一由本包做 AES-256-GCM 加密：
//
//	v1:<nonce_b64>:<ciphertext_b64>
//
// 密钥取自共享环境变量 MODEL_SECRET_KEY（与 JWT_SECRET 同级，无前缀），
// 长度必须为 32 字节。GCM 自带鉴权标签，密文被篡改会在解密时暴露。
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// KeySize AES-256 主密钥长度（字节）。
	KeySize = 32
	// CiphertextPrefix 密文版本前缀。将来轮换算法时按前缀识别旧数据。
	CiphertextPrefix = "v1:"
	// HintLength 脱敏提示保留的字符数。
	HintLength = 4
	// HintMaxBytes 与 model_provider.api_key_hint 列宽（varchar(8)）对齐，
	// 避免非 ASCII 密钥按字符取 4 位后仍超宽导致写入失败。
	HintMaxBytes = 8
)

var (
	// ErrInvalidKeySize 主密钥长度不符。
	ErrInvalidKeySize = errors.New("模型加密密钥长度必须为 32 字节")
	// ErrInvalidCiphertext 密文格式非法（缺少分隔符或段内容无法解码）。
	ErrInvalidCiphertext = errors.New("密文格式非法")
	// ErrDecryptFailed 解密失败，通常是密钥已轮换或密文被篡改。
	ErrDecryptFailed = errors.New("解密失败，密钥可能已更换或密文已损坏")
)

// Encrypter 基于 AES-256-GCM 的加解密器。
//
// 实例无内部可变状态，可全局复用；密钥只在构造时校验一次。
type Encrypter struct {
	aead cipher.AEAD
}

// NewEncrypter 用主密钥构造加解密器，密钥长度必须恰好为 KeySize。
//
// 长度不符时直接返回错误，由 main 在启动阶段 fail fast：长度自适应的做法
// （如哈希派生）会把「密钥配错」变成「静默得到另一把可用密钥」，
// 反而让轮换后的密文无法解密这件事更难排查。
func NewEncrypter(key string) (*Encrypter, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w：当前 %d 字节", ErrInvalidKeySize, len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("初始化 AES 加密块失败: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化 GCM 失败: %w", err)
	}

	return &Encrypter{aead: aead}, nil
}

// Encrypt 加密明文，返回带版本前缀的密文。每次调用使用新随机 nonce，
// 因此同一明文两次加密结果不同。
func (e *Encrypter) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成随机 nonce 失败: %w", err)
	}

	sealed := e.aead.Seal(nil, nonce, []byte(plaintext), nil)

	return CiphertextPrefix +
		base64.RawURLEncoding.EncodeToString(nonce) + ":" +
		base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密密文。密文不以 CiphertextPrefix 开头时按格式非法处理，
// 便于区分「版本不认识」与「解密失败」。
func (e *Encrypter) Decrypt(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, CiphertextPrefix) {
		return "", ErrInvalidCiphertext
	}
	body := strings.TrimPrefix(ciphertext, CiphertextPrefix)

	nonceB64, sealedB64, ok := strings.Cut(body, ":")
	if !ok {
		return "", ErrInvalidCiphertext
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("%w：nonce 解码失败", ErrInvalidCiphertext)
	}
	sealed, err := base64.RawURLEncoding.DecodeString(sealedB64)
	if err != nil {
		return "", fmt.Errorf("%w：密文解码失败", ErrInvalidCiphertext)
	}

	plaintext, err := e.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		// 不返回底层错误：GCM 的认证失败细节对调用方无用，
		// 且不应把密钥相关的内部状态透出边界。
		return "", ErrDecryptFailed
	}

	return string(plaintext), nil
}

// Hint 返回明文的脱敏提示（末 HintLength 个字符），是唯一允许外泄的片段。
//
// 空明文返回空串：此时 api_key_hint 列留空，前端据此显示「未配置」而不是 ****。
func Hint(plaintext string) string {
	if plaintext == "" {
		return ""
	}

	runes := []rune(plaintext)
	if len(runes) <= HintLength {
		return clampHint(string(runes))
	}

	return clampHint(string(runes[len(runes)-HintLength:]))
}

// clampHint 按字节裁剪提示，保证不超过 api_key_hint 列宽。
func clampHint(hint string) string {
	if len(hint) <= HintMaxBytes {
		return hint
	}
	// 超宽只可能是非 ASCII：按字符逐个收缩到列宽以内
	for len(hint) > HintMaxBytes {
		_, size := utf8.DecodeLastRuneInString(hint)
		hint = hint[:len(hint)-size]
	}
	return hint
}
