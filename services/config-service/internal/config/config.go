// Package config 负责从环境变量加载 config-service 配置。
//
// 配置集中存放于 config-service 服务目录的 .env 文件，使用 CONFIG_ 前缀；
// 系统环境变量优先级高于文件，缺失时回退默认值。
// 数据库与加密密钥属于共享配置（不带前缀），与其他服务保持同一套键名。
package config

import (
	"fmt"

	"github.com/museflow/pkg/envloader"
	"github.com/museflow/pkg/logger"
)

// Config config-service 运行配置。
type Config struct {
	Port  string // gRPC 监听端口
	DBDSN string // PostgreSQL 连接串（由共享 DB_* 配置拼装）
	Log   *logger.Config
	// SecretKey 模型凭证加密主密钥（共享键 MODEL_SECRET_KEY）。
	// 长度必须为 32 字节：api_key 与机密配置值都以它做 AES-256-GCM 加密，
	// 换密钥等于作废全部历史密文，因此启动时强校验，不做长度自适应。
	SecretKey string
}

// Load 读取 CONFIG_ 前缀配置并校验必填项。
//
// 数据库连接采用共享 DB_* 变量（所有服务共用），由这些变量拼装出 DSN，
// 与 user-service 的 config.Load 保持同一套拼装规则。
func Load() (*Config, error) {
	env := envloader.New("CONFIG", ".env")
	db := envloader.New("DB", ".env")

	cfg := &Config{
		Port:      env.Get("PORT", "5004"),
		DBDSN:     buildPostgresDSN(db),
		SecretKey: env.GetCommon("MODEL_SECRET_KEY", ""),
		Log:       loadLogConfig(env),
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("环境变量 MODEL_SECRET_KEY 未设置（模型凭证加密主密钥，须为 32 字节）")
	}

	return cfg, nil
}

// buildPostgresDSN 由分体参数拼装 PostgreSQL 连接串。
func buildPostgresDSN(db *envloader.Loader) string {
	host := db.GetCommon("DB_HOST", "localhost")
	port := db.GetCommonInt("DB_PORT", 5432)
	user := db.GetCommon("DB_USER", "postgres")
	password := db.GetCommon("DB_PASSWORD", "")
	name := db.GetCommon("DB_NAME", "museflow")

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, name)
}

// loadLogConfig 读取 LOG_ 前缀的日志配置，未显式设置时回退默认值。
//
// 日志统一输出到 stdout（由容器运行时 / k8s 收集），不落盘。
func loadLogConfig(env *envloader.Loader) *logger.Config {
	return &logger.Config{
		Level:   env.Get("LOG_LEVEL", "info"),
		Format:  env.Get("LOG_FORMAT", "json"),
		Console: env.GetBool("LOG_CONSOLE", true),
	}
}
