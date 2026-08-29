// Package config 负责从环境变量加载 user-service 配置。
//
// 配置集中存放于仓库根目录 .env 文件，使用 USER_ 前缀；
// 系统环境变量优先级高于文件，缺失时回退默认值。
package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/museflow/pkg/envloader"
	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/service/notify"
)

// Config user-service 运行配置。
type Config struct {
	Port      string // gRPC 监听端口
	DBDSN     string // PostgreSQL 连接串（由公共 DB_* 配置拼装）
	RedisAddr string // Redis 地址，用于 refresh 白名单与 access 黑名单（公共 REDIS_* 配置）
	RedisPass string
	RedisDB   int

	JWTSecret  string        // JWT 签名密钥（与 gateway 共享的 JWT_SECRET）
	AccessTTL  time.Duration // access token 有效期
	RefreshTTL time.Duration // refresh token 有效期
	BcryptCost int           // bcrypt 加密成本

	// 验证码与邮件（密码重置）
	CodeTTL       time.Duration // 验证码有效期
	CodeLength    int           // 验证码位数
	CodeResendCD  time.Duration // 重发冷却时间，防止短时间重复发送
	SMTP          notify.SMTPConfig // 邮件发送配置，未配置主机时降级为日志模式

	Log *logger.Config // 日志配置（由 LOG_ 前缀读取）
}

// Load 读取 USER_ 前缀配置并校验必填项。
//
// 数据库连接采用公共 DB_* 变量（所有服务共用），
// 由这些变量拼装出 DSN；同样支持「服务 .env 覆盖根 .env」的分层加载。
func Load() (*Config, error) {
	env := envloader.New("USER", ".env")
	db := envloader.New("DB", ".env")
	redis := envloader.New("REDIS", ".env")

	dbHost := db.GetCommon("DB_HOST", "localhost")
	dbPort := db.GetCommonInt("DB_PORT", 5432)
	dbUser := db.GetCommon("DB_USER", "postgres")
	dbPass := db.GetCommon("DB_PASSWORD", "")
	dbName := db.GetCommon("DB_NAME", "museflow")

	dsn := buildPostgresDSN(dbHost, dbPort, dbUser, dbPass, dbName)

	cfg := &Config{
		Port:       env.Get("PORT", "5002"),
		DBDSN:      dsn,
		RedisAddr:  redis.GetCommon("REDIS_ADDR", "localhost:6379"),
		RedisPass:  redis.GetCommon("REDIS_PASSWORD", ""),
		RedisDB:    redis.GetCommonInt("REDIS_DB", 0),
		JWTSecret:  env.GetCommon("JWT_SECRET", ""),
		AccessTTL:  env.GetDuration("ACCESS_TTL_SECONDS", 3600*time.Second),
		RefreshTTL: env.GetDuration("REFRESH_TTL_SECONDS", 2592000*time.Second),
		BcryptCost: env.GetInt("BCRYPT_COST", 10),
		// 验证码默认 6 位、10 分钟有效、60 秒内不可重发
		CodeTTL:      env.GetDuration("CODE_TTL_SECONDS", 600*time.Second),
		CodeLength:   env.GetInt("CODE_LENGTH", 6),
		CodeResendCD: env.GetDuration("CODE_RESEND_COOLDOWN_SECONDS", 60*time.Second),
		SMTP: notify.SMTPConfig{
			Host:     env.Get("SMTP_HOST", ""),
			Port:     env.GetInt("SMTP_PORT", 587),
			Username: env.Get("SMTP_USERNAME", ""),
			Password: env.Get("SMTP_PASSWORD", ""),
			From:     env.Get("SMTP_FROM", ""),
		},
		Log: loadLogConfig(env),
	}

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("环境变量 DB_HOST/DB_NAME 未设置，无法拼装数据库连接")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("环境变量 JWT_SECRET 未设置")
	}

	return cfg, nil
}

// buildPostgresDSN 由分体参数拼装 PostgreSQL 连接串。
func buildPostgresDSN(host string, port int, user, password, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbName)
}

// loadLogConfig 读取 LOG_ 前缀的日志配置，未显式设置时回退默认值。
func loadLogConfig(env *envloader.Loader) *logger.Config {
	compress := true
	if v := env.Get("LOG_COMPRESS", ""); v != "" {
		compress, _ = strconv.ParseBool(v)
	}
	return &logger.Config{
		Level:      env.Get("LOG_LEVEL", "info"),
		Format:     env.Get("LOG_FORMAT", "json"),
		OutputPath: env.Get("LOG_OUTPUT_PATH", "./logs"),
		ServiceName: env.Get("LOG_SERVICE_NAME", "user-service"),
		Console:    env.GetBool("LOG_CONSOLE", true),
		MaxSize:    env.GetInt("LOG_MAX_SIZE", 0),
		MaxBackups: env.GetInt("LOG_MAX_BACKUPS", 0),
		MaxAge:     env.GetInt("LOG_MAX_AGE", 0),
		Compress:   compress,
	}
}
