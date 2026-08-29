package errcode

import (
	"strconv"
	"strings"
)

// Lang 响应消息语言。
type Lang string

const (
	// LangZH 简体中文（默认）。
	LangZH Lang = "zh"
	// LangEN 英文。
	LangEN Lang = "en"
)

// messages 业务码到中英文提示信息的映射。
var messages = map[Code]map[Lang]string{
	CodeSuccess: {
		LangZH: "success",
		LangEN: "success",
	},
	CodeCreated: {
		LangZH: "创建成功",
		LangEN: "created",
	},
	CodeAccepted: {
		LangZH: "请求已接受",
		LangEN: "accepted",
	},

	CodeParamInvalid: {
		LangZH: "参数校验失败",
		LangEN: "invalid parameters",
	},
	CodeUnauthorized: {
		LangZH: "未认证或令牌已失效",
		LangEN: "unauthorized or token expired",
	},
	CodeForbidden: {
		LangZH: "无权限访问",
		LangEN: "permission denied",
	},
	CodeNotFound: {
		LangZH: "资源不存在",
		LangEN: "resource not found",
	},
	CodeConflict: {
		LangZH: "资源冲突",
		LangEN: "resource conflict",
	},
	CodeTooManyRequests: {
		LangZH: "请求过于频繁，请稍后再试",
		LangEN: "too many requests, please retry later",
	},
	CodeInternal: {
		LangZH: "服务内部错误",
		LangEN: "internal server error",
	},
	CodeServiceUnavail: {
		LangZH: "认证服务暂不可用",
		LangEN: "authentication service unavailable",
	},

	CodeUserNotFound: {
		LangZH: "用户不存在",
		LangEN: "user not found",
	},
	CodeWrongPassword: {
		LangZH: "邮箱或密码错误",
		LangEN: "invalid email or password",
	},
	CodeEmailRegistered: {
		LangZH: "邮箱已被注册",
		LangEN: "email already registered",
	},
	CodeTokenInvalid: {
		LangZH: "令牌无效或已过期",
		LangEN: "token invalid or expired",
	},
	CodeTokenExpired: {
		LangZH: "令牌已过期",
		LangEN: "token expired",
	},
	CodeRefreshInvalid: {
		LangZH: "刷新令牌无效或已过期",
		LangEN: "refresh token invalid or expired",
	},
	CodeDeviceMismatch: {
		LangZH: "设备校验失败",
		LangEN: "device verification failed",
	},
	CodeMissingToken: {
		LangZH: "缺少访问令牌",
		LangEN: "missing access token",
	},
	CodeMissingRefresh: {
		LangZH: "缺少刷新令牌",
		LangEN: "missing refresh token",
	},
	CodeMissingDevice: {
		LangZH: "缺少设备标识",
		LangEN: "missing device identifier",
	},
	CodeAccountLocked: {
		LangZH: "账号已被锁定，请稍后再试",
		LangEN: "account locked, please retry later",
	},
	CodeAccountDisabled: {
		LangZH: "账号已冻结或注销",
		LangEN: "account disabled or deleted",
	},
	CodeCodeInvalid: {
		LangZH: "邮箱验证码错误",
		LangEN: "invalid email verification code",
	},
	CodeEmailNotVerified: {
		LangZH: "邮箱未验证",
		LangEN: "email not verified",
	},
}

// defaultLang 默认语言（中文）。
const defaultLang = LangZH

// fallbackMessages 未定义业务码时的兜底提示。
var fallbackMessages = map[Lang]string{
	LangZH: "未知错误",
	LangEN: "unknown error",
}

// Message 返回业务码对应语言的提示信息，未定义时返回该语言的兜底文案。
// lang 为空或非法时回退到默认语言（中文）。
func Message(code Code, lang Lang) string {
	if lang != LangZH && lang != LangEN {
		lang = defaultLang
	}
	if m, ok := messages[code]; ok {
		if msg, ok := m[lang]; ok && msg != "" {
			return msg
		}
	}
	if msg, ok := fallbackMessages[lang]; ok {
		return msg
	}
	return fallbackMessages[defaultLang]
}

// ParseAcceptLanguage 从 HTTP Accept-Language 头解析目标语言。
// 支持 "zh-CN,zh;q=0.9,en;q=0.8" 这类带质量因子的格式，
// 命中 en*/EN 时返回英文，其余（含空、zh* 或无法识别）返回默认中文。
func ParseAcceptLanguage(header string) Lang {
	if header == "" {
		return defaultLang
	}

	parts := strings.Split(header, ",")
	best := defaultLang
	bestScore := -1.0
	for _, p := range parts {
		seg := strings.SplitN(p, ";", 2)
		tag := strings.TrimSpace(strings.ToLower(seg[0]))
		q := 1.0
		if len(seg) == 2 {
			kv := strings.SplitN(strings.TrimSpace(seg[1]), "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "q" {
				if v, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64); err == nil {
					q = v
				} else {
					q = 1.0
				}
			} else {
				q = 1.0
			}
		}
		if q <= bestScore {
			continue
		}
		if strings.HasPrefix(tag, "en") {
			best = LangEN
			bestScore = q
		} else if strings.HasPrefix(tag, "zh") {
			best = LangZH
			bestScore = q
		}
	}
	return best
}
