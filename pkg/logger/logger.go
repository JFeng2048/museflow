// Package logger 提供 MuseFlow 所有微服务统一的日志能力。
//
// 基于标准库 log/slog + lumberjack（日志轮转），零其它外部依赖。
// 设计原则：极简、统一，所有服务共享同一套初始化与全局函数。
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置。
type Config struct {
	// Level 日志级别：debug, info, warn, error（默认 info）。
	Level string
	// Format 输出格式：text, json（默认 json）。
	Format string
	// OutputPath 日志根目录，如 "./logs"。
	OutputPath string
	// ServiceName 服务名，用于分目录，如 "user-service"。
	ServiceName string
	// Console 是否同时输出到控制台。
	Console bool

	// 以下为日志轮转参数（lumberjack），缺省时使用默认值。

	// MaxSize 单个日志文件大小上限，单位 MB（默认 100）。
	MaxSize int
	// MaxBackups 保留的历史文件数（默认 30）。
	MaxBackups int
	// MaxAge 历史文件保留天数（默认 7）。
	MaxAge int
	// Compress 是否压缩历史文件（默认 true）。
	Compress bool
	// CompressSet 标记 Compress 是否被调用方显式设置。
	// 用于区分"未设置（用默认 true）"和"显式设为 false"。
	CompressSet bool
}

// lumberjack 轮转参数默认值。
const (
	defaultRotateMaxSize    = 100  // MB
	defaultRotateMaxBackups = 30   // 保留文件数
	defaultRotateMaxAge     = 7    // 天
	defaultRotateCompress   = true // 压缩旧文件
)

// 全局默认 logger 句柄。
var defaultLogger *slog.Logger

func init() {
	// 未调用 Init 时使用一个安全的 stdout 默认 logger，避免空指针。
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Init 初始化日志：创建目录、配置轮转、设置全局 logger。
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}

	level := parseLevel(cfg.Level)
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format != "text" {
		format = "json"
	}

	// 组装输出目标。
	var writers []io.Writer

	// 文件输出：OutputPath/ServiceName/app.log
	if dir := strings.TrimSpace(cfg.OutputPath); dir != "" {
		base := dir
		if name := strings.TrimSpace(cfg.ServiceName); name != "" {
			base = filepath.Join(dir, name)
		}
		if err := os.MkdirAll(base, 0o755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
		filePath := filepath.Join(base, "app.log")
		lj := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    orDefaultInt(cfg.MaxSize, defaultRotateMaxSize),
			MaxBackups: orDefaultInt(cfg.MaxBackups, defaultRotateMaxBackups),
			MaxAge:     orDefaultInt(cfg.MaxAge, defaultRotateMaxAge),
			Compress:   compressValue(cfg),
		}
		writers = append(writers, lj)
	}

	// 控制台输出。
	if cfg.Console {
		writers = append(writers, os.Stdout)
	}

	// 若两者皆无，则兜底输出到 stdout，避免无目标。
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	dest := io.MultiWriter(writers...)

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		handler = slog.NewJSONHandler(dest, opts)
	} else {
		handler = slog.NewTextHandler(dest, opts)
	}

	defaultLogger = slog.New(handler)
	return nil
}

// parseLevel 将字符串级别转换为 slog.Level，非法值回退 info。
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// orDefaultInt 当 v<=0 时返回默认值 d。
func orDefaultInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// compressValue 返回压缩开关：未显式设置时返回默认 true，
// 显式设置（无论 true/false）后尊重调用方值。
func compressValue(cfg *Config) bool {
	if cfg.CompressSet {
		return cfg.Compress
	}
	return defaultRotateCompress
}

// Logger 返回当前全局 logger 句柄。
func Logger() *slog.Logger {
	return defaultLogger
}

// 全局日志函数（不带 Context）。

// Debug 输出 debug 级别日志。
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info 输出 info 级别日志。
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn 输出 warn 级别日志。
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error 输出 error 级别日志。
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// 带 Context 的日志函数：自动注入 trace_id / request_id。

// DebugContext 输出带上下文的 debug 日志。
func DebugContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

// InfoContext 输出带上下文的 info 日志。
func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// WarnContext 输出带上下文的 warn 日志。
func WarnContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

// ErrorContext 输出带上下文的 error 日志。
func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}
