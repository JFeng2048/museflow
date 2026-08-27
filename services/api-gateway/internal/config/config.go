// Package config 负责从环境变量加载 api-gateway 配置。
//
// 配置集中存放于仓库根目录 .env 文件，使用 GATEWAY_ 前缀；
// 系统环境变量优先级高于文件，缺失时回退默认值。
package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/museflow/pkg/envloader"
	"github.com/museflow/pkg/logger"
)

// Config api-gateway 运行配置。
type Config struct {
	Port           string   // HTTP 监听端口
	UserServiceURL string   // user-service gRPC 地址
	JWTSecret      string   // 与 user-service 共享的 JWT 密钥，用于本地验签
	AllowOrigins   []string // CORS 允许的来源
	CookieSecure   bool     // Cookie 是否仅 HTTPS 传输（生产置 true）
	CookieSameSite string   // Cookie SameSite 策略: lax/strict/none
	CookieDomain   string   // Cookie 作用域

	Log *logger.Config // 日志配置（由 LOG_ 前缀读取）
}

// Load 读取 GATEWAY_ 前缀配置并校验必填项。
func Load() (*Config, error) {
	env := envloader.New("GATEWAY", ".env")

	cfg := &Config{
		Port:           env.Get("PORT", "5001"),
		UserServiceURL: env.Get("USER_SERVICE_URL", "localhost:5002"),
		JWTSecret:      env.GetCommon("JWT_SECRET", ""),
		AllowOrigins:   splitAndTrim(env.Get("ALLOW_ORIGINS", "http://localhost:5173")),
		CookieSecure:   env.GetBool("COOKIE_SECURE", false),
		CookieSameSite: env.Get("COOKIE_SAMESITE", "lax"),
		CookieDomain:   env.Get("COOKIE_DOMAIN", ""),
		Log:            loadLogConfig(env),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("环境变量 JWT_SECRET 未设置")
	}

	return cfg, nil
}

// loadLogConfig 读取 LOG_ 前缀的日志配置，未显式设置时回退默认值。
func loadLogConfig(env *envloader.Loader) *logger.Config {
	compress := true
	if v := env.Get("LOG_COMPRESS", ""); v != "" {
		compress, _ = strconv.ParseBool(v)
	}
	return &logger.Config{
		Level:       env.Get("LOG_LEVEL", "info"),
		Format:      env.Get("LOG_FORMAT", "json"),
		OutputPath:  env.Get("LOG_OUTPUT_PATH", "./logs"),
		ServiceName: env.Get("LOG_SERVICE_NAME", "api-gateway"),
		Console:     env.GetBool("LOG_CONSOLE", true),
		MaxSize:     env.GetInt("LOG_MAX_SIZE", 0),
		MaxBackups:  env.GetInt("LOG_MAX_BACKUPS", 0),
		MaxAge:      env.GetInt("LOG_MAX_AGE", 0),
		Compress:    compress,
	}
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
