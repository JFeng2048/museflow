package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/museflow/config-service/internal/pkg/secret"
)

// maxAPIKeyLen 密钥长度上限。上游 Key 最长见过几百字节，4096 足够宽松，
// 同时挡住把整段配置粘贴进来的误操作。
const maxAPIKeyLen = 4096

// apiKeySecret 加密后的密钥材料。
type apiKeySecret struct {
	// Ciphertext 落库密文，空串表示该渠道未配置密钥。
	Ciphertext string
	// Hint 末 4 位脱敏提示。
	Hint string
	// UpdatedAt 密钥写入时间，仅在真正轮换时非空。
	UpdatedAt *time.Time
}

// sealAPIKey 加密 API Key 并生成脱敏提示。
//
// 空 key 不加密、密文留空：本地部署的渠道（如 Ollama）本来就不需要密钥，
// 存一段空串密文会让「未配置」和「已配置但提示为空」难以区分。
// 更新场景也依赖这一点——调用方判断 Ciphertext 是否为空即可知道要不要写 key 列。
func (s *Service) sealAPIKey(plaintext string) (apiKeySecret, error) {
	key := strings.TrimSpace(plaintext)
	if key == "" {
		return apiKeySecret{}, nil
	}
	if len(key) > maxAPIKeyLen {
		return apiKeySecret{}, invalidArgumentf("api_key 长度不能超过 %d 个字符", maxAPIKeyLen)
	}

	ciphertext, err := s.secrets.Encrypt(key)
	if err != nil {
		return apiKeySecret{}, fmt.Errorf("加密 api_key 失败: %w", err)
	}

	now := time.Now()
	return apiKeySecret{Ciphertext: ciphertext, Hint: secret.Hint(key), UpdatedAt: &now}, nil
}

// sealSecretValue 加密通用配置里的机密值，与 api_key 走同一把主密钥。
//
// 与 api_key 的区别：配置值不允许为空，调用方必须先确认非空再进来。
func (s *Service) sealSecretValue(plaintext string) (string, error) {
	ciphertext, err := s.secrets.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("加密配置值失败: %w", err)
	}
	return ciphertext, nil
}
