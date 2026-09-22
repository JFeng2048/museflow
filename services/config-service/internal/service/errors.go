package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/museflow/config-service/internal/model"
)

// ErrInvalidArgument 请求参数不合法。
//
// 细分场景用 fmt.Errorf("%w：具体原因", ErrInvalidArgument) 包装：上层用
// errors.Is 统一映射成 codes.InvalidArgument，具体原因能直接透给调用方，
// 不必再维护一张错误码表。
var ErrInvalidArgument = errors.New("请求参数不合法")

// ErrCredentialUnreadable 渠道密钥无法解密（主密钥已轮换或密文被损坏）。
//
// 只在「拉取上游模型目录」这条路径上可能出现：其余接口从不回读密钥，
// 密文坏了也无从察觉。这里显式报错而不是当「未配置密钥」处理——后者会把
// 失败推给上游，最终只回一句 401，看起来像是密钥本身失效，排查方向就错了。
var ErrCredentialUnreadable = errors.New("渠道密钥无法解密，请重新填写 api_key")

// invalidArgumentf 构造带原因的 InvalidArgument 错误。
func invalidArgumentf(format string, args ...any) error {
	return fmt.Errorf("%w：%s", ErrInvalidArgument, fmt.Sprintf(format, args...))
}

// 字段长度上限，与 database/config_svc.sql 的列宽保持一致。
const (
	maxCodeLen         = 50  // model_provider.code
	maxNameLen         = 100 // 各处 name
	maxProtocolLen     = 50  // protocol
	maxBaseURLLen      = 500 // base_url
	maxAPIModelLen     = 100 // api_model
	maxOrganizationLen = 255 // organization
	maxConfigGroupLen  = 50  // system_setting.config_group
	maxSettingKeyLen   = 100 // system_setting.key
	// maxDescriptionLen 说明文字上限。description 是 text 列，数据库不限制长度，
	// 但用户端选择器只会显示前几十个字，放任长文本只会让后台表格难读。
	maxDescriptionLen = 2000

	// maxExtraEntries 协议扩展参数条数上限，与单条键值长度上限一起使用：
	// extra 是自由映射，没有约束时容易被当成第二个配置中心，越写越乱。
	maxExtraEntries  = 32
	maxExtraKeyLen   = 100
	maxExtraValueLen = 500
)

// 分页参数默认值与上限，与 api-gateway 的 normalizePage 保持一致。
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// normalizePage 归一分页参数。
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// pageOffset 由页码与每页数量换算偏移量。
func pageOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// validateText 校验文本字段并返回去空白后的值。
//
// required 为 false 时允许空值：可选字段（organization / description）空表示不写，
// 由调用方决定是跳过该列还是置 NULL。
func validateText(field, value string, maxLen int, required bool) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		if required {
			return "", invalidArgumentf("%s不能为空", field)
		}
		return "", nil
	}
	if len(v) > maxLen {
		return "", invalidArgumentf("%s长度不能超过 %d 个字符", field, maxLen)
	}
	return v, nil
}

// optionalText 校验可选文本字段，空值返回 nil，由 GORM 落成 NULL。
func optionalText(field, value string, maxLen int) (*string, error) {
	v, err := validateText(field, value, maxLen, false)
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, nil
	}
	return &v, nil
}

// validateID 校验主键入参。
//
// id <= 0 一律按参数非法处理：上游是路径参数，出现 0 或负数只可能是调用方拼错了
// URL，此时返回 NotFound 会让人以为资源真的不存在。
func validateID(id int64) error {
	if id <= 0 {
		return invalidArgumentf("id 必须大于 0")
	}
	return nil
}

// validateUserUUID 校验用户归属标识。
//
// 用户级表（user_model_provider / user_model）全部按 user_uuid 隔离，
// 空 UUID 只可能是调用方没带上登录态。此时放行会让数据落到 uuid.Nil 名下，
// 事后谁也查不回来，因此在入口就拦掉。
func validateUserUUID(userUUID uuid.UUID) error {
	if userUUID == uuid.Nil {
		return invalidArgumentf("user_uuid 不能为空")
	}
	return nil
}

// validateNonNegative 校验非负计数字段（上下文窗口、积分、token 上限等）。
func validateNonNegative(field string, value int32) error {
	if value < 0 {
		return invalidArgumentf("%s不能为负数", field)
	}
	return nil
}

// knownValues 把合法取值拼成错误提示，如 "chat / embedding / rerank"。
func knownValues(values ...string) string {
	return strings.Join(values, " / ")
}

// validateProtocol 校验并归一化通讯协议（统一小写）。
func validateProtocol(protocol string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(protocol))
	if value == "" {
		return "", invalidArgumentf("protocol 不能为空")
	}

	switch value {
	case model.ProtocolOpenAI, model.ProtocolAnthropic, model.ProtocolGemini, model.ProtocolCustom:
		return value, nil
	default:
		return "", invalidArgumentf("protocol 仅支持 %s",
			knownValues(model.ProtocolOpenAI, model.ProtocolAnthropic, model.ProtocolGemini, model.ProtocolCustom))
	}
}

// normalizeExtra 校验并归一化协议扩展参数（键去空白，空键视为非法）。
//
// 空入参返回 nil：GORM 的 StringMap.Value() 对空映射写 NULL，
// 与「没有扩展参数」这一语义一致。
func normalizeExtra(extra map[string]string) (map[string]string, error) {
	if len(extra) == 0 {
		return nil, nil
	}
	if len(extra) > maxExtraEntries {
		return nil, invalidArgumentf("extra 最多支持 %d 项", maxExtraEntries)
	}

	out := make(map[string]string, len(extra))
	for k, v := range extra {
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, invalidArgumentf("extra 的键不能为空")
		}
		if len(key) > maxExtraKeyLen {
			return nil, invalidArgumentf("extra 的键长度不能超过 %d 个字符", maxExtraKeyLen)
		}
		if len(v) > maxExtraValueLen {
			return nil, invalidArgumentf("extra 的值长度不能超过 %d 个字符", maxExtraValueLen)
		}
		out[key] = v
	}
	return out, nil
}

// validateModelType 校验并归一化模型类型。
//
// 做白名单校验而不是放任自由文本：model_type 同时驱动用户端的筛选器与后续生成
// 服务的调用分支，写错一个字母（如 embeding）不会报错，只会静默产生一个谁也刷
// 不出来的模型。
func validateModelType(modelType string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(modelType))
	if value == "" {
		return "", invalidArgumentf("model_type 不能为空")
	}

	switch value {
	case model.ModelTypeChat, model.ModelTypeEmbedding, model.ModelTypeRerank,
		model.ModelTypeVision, model.ModelTypeImage, model.ModelTypeAudio, model.ModelTypeVideo:
		return value, nil
	default:
		return "", invalidArgumentf("model_type 仅支持 %s", knownValues(
			model.ModelTypeChat, model.ModelTypeEmbedding, model.ModelTypeRerank,
			model.ModelTypeVision, model.ModelTypeImage, model.ModelTypeAudio, model.ModelTypeVideo))
	}
}

// validateValueType 校验并归一化配置值类型。
//
// 空值按 string 处理：后台表单不填类型时最常见的意图就是存一段文本。
func validateValueType(valueType string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(valueType))
	if value == "" {
		return model.ValueTypeString, nil
	}

	switch value {
	case model.ValueTypeString, model.ValueTypeNumber, model.ValueTypeBool, model.ValueTypeJSON:
		return value, nil
	default:
		return "", invalidArgumentf("value_type 仅支持 %s", knownValues(
			model.ValueTypeString, model.ValueTypeNumber, model.ValueTypeBool, model.ValueTypeJSON))
	}
}

// validateBaseURL 校验并归一化 API 基础地址。
//
// 归一化只做两件事：去首尾空白、去结尾斜杠。结尾斜杠是最常见的配置错误——
// 拼路径时 "/v1/" + "/chat/completions" 会成双斜杠，部分上游直接 404。
func validateBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", invalidArgumentf("base_url 不能为空")
	}
	if len(value) > maxBaseURLLen {
		return "", invalidArgumentf("base_url 长度不能超过 %d 个字符", maxBaseURLLen)
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return "", invalidArgumentf("base_url 不是合法地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", invalidArgumentf("base_url 只支持 http / https 协议")
	}
	if parsed.Host == "" {
		return "", invalidArgumentf("base_url 缺少主机名")
	}

	return strings.TrimRight(value, "/"), nil
}
