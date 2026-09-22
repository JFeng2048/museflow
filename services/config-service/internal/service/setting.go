package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/repository"
)

// maxSecretValueLen 机密配置值长度上限，与 maxAPIKeyLen 取同一量级：
// 两者的本质都是「上游凭证」，放开长度只会容纳误粘贴的整段配置。
const maxSecretValueLen = 4096

// SettingInput 通用系统配置写入参数。
//
// Value 与 SecretValue 二选一，由 IsSecret 决定落哪一列：数据库上有 CHECK
// 约束强制两者恰好一个非空，在 service 层先校验，是为了把「填错字段」
// 变成可读的 InvalidArgument，而不是一条 500。
type SettingInput struct {
	ConfigGroup string
	Key         string
	Value       string
	SecretValue string
	IsSecret    bool
	ValueType   string
	IsPublic    bool
	Description string
}

// SettingResult 写入结果。
//
// SecretEcho 是机密配置的明文，只在写入那一刻有意义：调用方必须只在写入响应里
// 透出它，读取路径（List / Get）一律留空，前端据此把已有机密显示为「已配置」。
type SettingResult struct {
	Setting    *model.SystemSetting
	SecretEcho string
}

// ListSettings 查询系统配置。
//
// config_group 传空表示全部分组；超长按参数非法返回，而不是静默查不到——
// 拼错的分组名与「确实没有配置」在调用方眼里应该能区分开。
func (s *Service) ListSettings(ctx context.Context, configGroup string, onlyPublic bool) ([]model.SystemSetting, error) {
	group, err := validateText("config_group", configGroup, maxConfigGroupLen, false)
	if err != nil {
		return nil, err
	}
	return s.settings.List(ctx, repository.SettingListFilter{
		ConfigGroup: group,
		OnlyPublic:  onlyPublic,
	})
}

// GetSetting 按 (config_group, key) 查询配置。
func (s *Service) GetSetting(ctx context.Context, configGroup, key string) (*model.SystemSetting, error) {
	group, err := validateText("config_group", configGroup, maxConfigGroupLen, true)
	if err != nil {
		return nil, err
	}
	settingKey, err := validateText("key", key, maxSettingKeyLen, true)
	if err != nil {
		return nil, err
	}
	return s.settings.Get(ctx, group, settingKey)
}

// UpsertSetting 写入配置，存在则更新、不存在则创建。
//
// 机密值留空表示沿用旧密文：后台改一句说明文字不该逼着重新填一遍密钥。
// 这里取的是既有密文而非解密结果——本服务没有任何路径需要知道明文，
// 也就没有理由为此持有解密能力之外的特权。
func (s *Service) UpsertSetting(ctx context.Context, input SettingInput) (*SettingResult, error) {
	group, err := validateText("config_group", input.ConfigGroup, maxConfigGroupLen, true)
	if err != nil {
		return nil, err
	}
	settingKey, err := validateText("key", input.Key, maxSettingKeyLen, true)
	if err != nil {
		return nil, err
	}
	valueType, err := validateValueType(input.ValueType)
	if err != nil {
		return nil, err
	}
	description, err := optionalText("description", input.Description, maxDescriptionLen)
	if err != nil {
		return nil, err
	}

	secretPlaintext := strings.TrimSpace(input.SecretValue)
	plaintext := strings.TrimSpace(input.Value)
	if len(secretPlaintext) > maxSecretValueLen {
		return nil, invalidArgumentf("secret_value 长度不能超过 %d 个字符", maxSecretValueLen)
	}
	if len(plaintext) > maxDescriptionLen {
		// 非机密值同样是配置文本，复用说明文字的上限：
		// 超长内容应该走对象存储或单独的服务，不该塞进一张 key-value 表。
		return nil, invalidArgumentf("value 长度不能超过 %d 个字符", maxDescriptionLen)
	}

	setting := &model.SystemSetting{
		ConfigGroup: group,
		Key:         settingKey,
		IsSecret:    input.IsSecret,
		ValueType:   valueType,
		IsPublic:    input.IsPublic,
		Description: description,
	}

	var echo string
	if input.IsSecret {
		if plaintext != "" {
			return nil, invalidArgumentf("机密配置不能同时提供 value，请只填 secret_value")
		}
		if secretPlaintext == "" {
			ciphertext, err := s.existingSecretCiphertext(ctx, group, settingKey)
			if err != nil {
				return nil, err
			}
			setting.SecretValue = ciphertext
		} else {
			ciphertext, err := s.sealSecretValue(secretPlaintext)
			if err != nil {
				return nil, err
			}
			setting.SecretValue = &ciphertext
			echo = secretPlaintext
		}
	} else {
		if secretPlaintext != "" {
			return nil, invalidArgumentf("非机密配置不能提供 secret_value，需要加密请把 is_secret 设为 true")
		}
		encoded, err := encodeSettingValue(valueType, plaintext)
		if err != nil {
			return nil, err
		}
		setting.Value = model.JSONText(encoded)
	}

	if err := s.settings.Upsert(ctx, setting); err != nil {
		return nil, err
	}
	return &SettingResult{Setting: setting, SecretEcho: echo}, nil
}

// DeleteSetting 按 (config_group, key) 删除配置。
func (s *Service) DeleteSetting(ctx context.Context, configGroup, key string) error {
	group, err := validateText("config_group", configGroup, maxConfigGroupLen, true)
	if err != nil {
		return err
	}
	settingKey, err := validateText("key", key, maxSettingKeyLen, true)
	if err != nil {
		return err
	}
	return s.settings.Delete(ctx, group, settingKey)
}

// existingSecretCiphertext 取既有机密配置的密文，用于「不改密钥」的更新。
//
// 三种拿不到密文的情况都归成参数非法：配置不存在、配置原本不是机密、
// 以及配置是机密但密文为空（脏数据）。此时调用方必须显式提供新密钥，
// 否则写进去的就是一条违反 CHECK 约束的行。
func (s *Service) existingSecretCiphertext(ctx context.Context, group, key string) (*string, error) {
	existing, err := s.settings.Get(ctx, group, key)
	if err != nil {
		if errors.Is(err, repository.ErrSettingNotFound) {
			return nil, invalidArgumentf("新增机密配置必须提供 secret_value")
		}
		return nil, err
	}
	if !existing.IsSecret || existing.SecretValue == nil || *existing.SecretValue == "" {
		return nil, invalidArgumentf("该配置原本不是机密配置，改为机密配置时必须提供 secret_value")
	}
	return existing.SecretValue, nil
}

// encodeSettingValue 按 value_type 把入参序列化成 jsonb 文本。
//
// value 列是 jsonb，存不了裸字符串，所以 string 也要包一层引号。
// 读取端拿到的统一是「JSON 序列化后的字符串」，按 value_type 解析即可，
// 服务端不负责替调用方拆引号——那会让读写两侧的语义不对称。
func encodeSettingValue(valueType, value string) (string, error) {
	if value == "" {
		return "", invalidArgumentf("非机密配置必须提供 value")
	}

	switch valueType {
	case model.ValueTypeString:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("序列化配置值失败: %w", err)
		}
		return string(encoded), nil

	case model.ValueTypeNumber:
		if err := validateJSONNumber(value); err != nil {
			return "", err
		}
		return value, nil

	case model.ValueTypeBool:
		if value != "true" && value != "false" {
			return "", invalidArgumentf("value_type 为 bool 时 value 只能是 true 或 false")
		}
		return value, nil

	default: // model.ValueTypeJSON
		if !json.Valid([]byte(value)) {
			return "", invalidArgumentf("value_type 为 json 时 value 必须是合法 JSON")
		}
		// 压缩掉无关空白，同一份配置不会因为格式化差异被当成两次不同写入
		var buf bytes.Buffer
		if err := json.Compact(&buf, []byte(value)); err != nil {
			return "", invalidArgumentf("value_type 为 json 时 value 必须是合法 JSON")
		}
		return buf.String(), nil
	}
}

// validateJSONNumber 校验入参是 JSON 认可的数字。
//
// strconv.ParseFloat 会放过 NaN / Inf 与十六进制浮点字面量，这些都不是合法
// JSON，写进 jsonb 列才会报错，因此在这里再叠一层 json.Valid。
func validateJSONNumber(value string) error {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || !json.Valid([]byte(value)) {
		return invalidArgumentf("value_type 为 number 时 value 必须是数字")
	}
	return nil
}
