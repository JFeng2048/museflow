// Package turnstile 封装 Cloudflare Turnstile 的服务端校验（siteverify）。
//
// 完整链路：前端渲染 widget → 用户通过验证拿到一次性 token → 业务请求携带 token →
// 三个容易踩的坑，实现里都有对应处理：
//  1. token 一次性：无论核验成功与否，同一 token 再次提交都会被判为 timeout-or-duplicate。
//     因此前端每次发送都必须重新取 token，不能缓存复用。
//  2. 必须服务端校验：前端拿到 token 不代表校验通过，只有 siteverify 返回 success=true 才算数。
//  3. 未配置 Secret 时降级为「跳过校验」以便本地开发；生产环境未配置等同于无防护，
//     此时校验仍会放行，但会输出告警日志。
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/museflow/pkg/logger"
)

// DefaultEndpoint Cloudflare siteverify 官方地址。
const DefaultEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// 面向用户的错误。文案可直接展示，不暴露内部细节。
var (
	// ErrTokenMissing 请求未携带人机验证令牌。
	ErrTokenMissing = errors.New("请先完成人机验证")
	// ErrTokenInvalid 令牌无效、已过期或已被使用过。
	ErrTokenInvalid = errors.New("人机验证未通过，请重新完成验证")
	// ErrServiceUnavailable 校验服务不可用（网络错误或 Cloudflare 侧故障）。
	ErrServiceUnavailable = errors.New("人机验证服务暂时不可用，请稍后重试")
	// ErrActionMismatch 令牌的 action 与本次请求场景不一致。
	ErrActionMismatch = errors.New("人机验证场景不匹配，请重新完成验证")
	// ErrHostnameMismatch 令牌的 hostname 不在允许列表中。
	ErrHostnameMismatch = errors.New("人机验证来源不合法")
)

// errorMessages 把 Cloudflare 返回的错误码翻译成中文，仅用于日志排查。
//
// 完整列表见 https://developers.cloudflare.com/turnstile/troubleshooting/
var errorMessages = map[string]string{
	"missing-input-secret":   "缺少密钥配置",
	"invalid-input-secret":   "密钥无效或与站点不匹配",
	"missing-input-response": "缺少验证令牌",
	"invalid-input-response": "验证令牌无效或已过期",
	"bad-request":            "请求格式错误",
	"timeout-or-duplicate":   "验证令牌已过期或已被使用过（令牌一次性）",
	"internal-error":         "Cloudflare 内部错误",
}

// Config 人机验证配置。
type Config struct {
	// Secret 密钥（服务端专用，切勿下发到前端或提交到仓库）。
	// 为空时校验降级为「跳过」，仅适用于本地开发。
	Secret string
	// Endpoint siteverify 地址，为空时使用 DefaultEndpoint。
	Endpoint string
	// Timeout 单次校验超时。人机验证在发送验证码的链路上，超时必须短，
	// 否则慢校验会把接口的响应时间拖回去。
	Timeout time.Duration
	// AllowedHostnames 允许的来源域名白名单，为空表示不校验 hostname。
	// 生产环境建议填写，可防止密钥被别的站点盗用。
	AllowedHostnames []string
}

// Enabled 判断是否已配置密钥。
func (c Config) Enabled() bool { return c.Secret != "" }

// applyDefaults 补全默认值。
func (c *Config) applyDefaults() {
	c.Secret = strings.TrimSpace(strings.Trim(c.Secret, `"'`))
	if c.Endpoint == "" {
		c.Endpoint = DefaultEndpoint
	}
	if c.Timeout <= 0 {
		// 默认 15s：国内访问 challenges.cloudflare.com 常见 TLS/首包偏慢，
		// 5s 容易在「等待响应头」阶段直接超时。
		c.Timeout = 15 * time.Second
	}
}

// Client siteverify 客户端。
type Client struct {
	cfg Config
	hc  *http.Client
}

// New 构造校验客户端。
func New(cfg Config) *Client {
	cfg.applyDefaults()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = true
	transport.TLSHandshakeTimeout = min(10*time.Second, cfg.Timeout)
	transport.ResponseHeaderTimeout = cfg.Timeout
	return &Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// NewNoop 构造跳过校验的客户端（未配置密钥时使用）。
//
// 返回一个已禁用、Verify 恒返回 nil 的客户端，让调用方无需到处判空。
func NewNoop() *Client { return &Client{} }

// Verifier 校验抽象，便于单元测试注入替身。
type Verifier interface {
	// Verify 校验令牌。token 为空、无效或校验服务不可用时返回错误。
	// expectedAction 为空表示不校验 action。
	Verify(ctx context.Context, token, remoteIP, expectedAction string) error
}

// verifyResponse siteverify 响应体。
type verifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
	ErrorCodes  []string `json:"error-codes"`
}

// Verify 校验人机验证令牌。
//
// 未配置密钥时直接放行（降级），仅用于本地开发。
// 校验采用 fail-closed：网络错误或 Cloudflare 故障一律拒绝，绝不因抖动放行。
func (c *Client) Verify(ctx context.Context, token, remoteIP, expectedAction string) error {
	if !c.cfg.Enabled() {
		logger.WarnContext(ctx, "人机验证未配置（TURNSTILE_SECRET 为空），已跳过校验；生产环境必须配置以防止验证码接口被刷")
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrTokenMissing
	}

	// siteverify 使用表单编码。remoteip 仅提交公网地址：
	// 本地 ::1 / 127.0.0.1 会让 Cloudflare 直接回 HTTP 400。
	form := url.Values{
		"secret":   {c.cfg.Secret},
		"response": {token},
	}
	if ip := sanitizeRemoteIP(remoteIP); ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("构造人机验证请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MuseFlow-user-service/turnstile")

	resp, err := c.hc.Do(req)
	if err != nil {
		logger.WarnContext(ctx, "调用人机验证服务失败",
			logger.Err(err),
			"endpoint", c.cfg.Endpoint,
			"timeout", c.cfg.Timeout.String(),
			"hint", "多为访问 Cloudflare 超时：提高 USER_TURNSTILE_TIMEOUT_SECONDS，或为进程配置 HTTPS_PROXY；本地可先清空 USER_TURNSTILE_SECRET 跳过校验",
		)
		return ErrServiceUnavailable
	}
	defer resp.Body.Close()

	// 限制读取长度，避免异常响应耗尽内存
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return ErrServiceUnavailable
	}

	var out verifyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		logger.WarnContext(ctx, "解析人机验证响应失败",
			logger.Err(err),
			"status", resp.StatusCode,
			"body", truncateForLog(body),
		)
		return ErrServiceUnavailable
	}

	if !out.Success {
		logger.WarnContext(ctx, "人机验证未通过",
			"status", resp.StatusCode,
			"errors", strings.Join(describeErrorCodes(out.ErrorCodes), ", "),
			"hostname", out.Hostname,
			"action", out.Action,
		)
		for _, code := range out.ErrorCodes {
			switch code {
			case "invalid-input-secret", "missing-input-secret":
				logger.ErrorContext(ctx, "USER_TURNSTILE_SECRET 无效：请填 Cloudflare 控制台的 Secret Key，不要填 Site Key")
				return ErrServiceUnavailable
			case "timeout-or-duplicate", "invalid-input-response", "missing-input-response", "bad-request":
				return ErrTokenInvalid
			}
		}
		if resp.StatusCode != http.StatusOK {
			return ErrServiceUnavailable
		}
		return ErrTokenInvalid
	}

	if resp.StatusCode != http.StatusOK {
		logger.WarnContext(ctx, "人机验证服务返回异常状态码", "status", resp.StatusCode, "body", truncateForLog(body))
		return ErrServiceUnavailable
	}

	// action 校验：widget 未设置 action 时为空，此时跳过（兼容未传 action 的旧客户端）
	if expectedAction != "" && out.Action != "" && out.Action != expectedAction {
		logger.WarnContext(ctx, "人机验证 action 不匹配", "want", expectedAction, "got", out.Action)
		return ErrActionMismatch
	}

	// hostname 白名单：未配置时不校验
	if len(c.cfg.AllowedHostnames) > 0 && !containsHostname(c.cfg.AllowedHostnames, out.Hostname) {
		logger.WarnContext(ctx, "人机验证 hostname 不在白名单", "hostname", out.Hostname)
		return ErrHostnameMismatch
	}

	return nil
}

// containsHostname 判断 hostname 是否在白名单中（忽略大小写与末尾点）。
func containsHostname(allowed []string, hostname string) bool {
	host := strings.TrimSuffix(strings.ToLower(hostname), ".")
	for _, a := range allowed {
		if strings.TrimSuffix(strings.ToLower(a), ".") == host {
			return true
		}
	}
	return false
}

// describeErrorCodes 把错误码翻译成中文，用于日志。
func describeErrorCodes(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if msg, ok := errorMessages[c]; ok {
			out = append(out, c+"="+msg)
			continue
		}
		out = append(out, c)
	}
	return out
}

// sanitizeRemoteIP 把网关传来的地址整理成 siteverify 可接受的公网 IP。
// 回环 / 内网 / 非法值一律省略，避免 Cloudflare 返回 HTTP 400。
func sanitizeRemoteIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	raw = strings.Trim(raw, "[]")
	if i := strings.IndexByte(raw, '%'); i >= 0 {
		raw = raw[:i]
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return ""
	}
	return ip.String()
}

func truncateForLog(body []byte) string {
	const max = 256
	s := strings.TrimSpace(string(body))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
