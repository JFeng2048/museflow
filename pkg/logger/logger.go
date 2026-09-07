// Package logger 提供 MuseFlow 所有微服务统一的日志能力。
//
// 基于标准库 log/slog，零外部依赖。日志统一输出到标准输出/错误流
// （stdout/stderr），不做文件落盘：容器日志由容器运行时 / k8s 收集。
// 设计原则：极简、统一，所有服务共享同一套初始化与全局函数。
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config 日志配置。
type Config struct {
	// Level 日志级别：debug, info, warn, error（默认 info）。
	Level string
	// Format 输出格式：text, json（默认 json）。
	Format string
	// Console 是否输出到 stdout（默认 true）；false 时输出到 stderr。
	Console bool
}

// 全局默认 logger 句柄。
var defaultLogger *slog.Logger

func init() {
	// 未调用 Init 时使用一个安全的 stdout 默认 logger，避免空指针。
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Init 初始化日志并设置全局 logger。
//
// 日志不落盘，统一写标准流：
//   - Console 为 true（默认）输出到 stdout；
//   - Console 为 false 输出到 stderr。
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}

	level := parseLevel(cfg.Level)
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format != "text" {
		format = "json"
	}

	var dest io.Writer = os.Stdout
	if !cfg.Console {
		dest = os.Stderr
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
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
