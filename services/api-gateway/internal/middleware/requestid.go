package middleware

import (
	"crypto/rand"
	"encoding/hex"
)

// newRequestID 生成 16 字节的随机十六进制请求标识。
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极罕见的失败兜底，使用空串也不会破坏日志
		return ""
	}
	return hex.EncodeToString(b)
}
