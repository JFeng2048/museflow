package logger

import "log/slog"

// 常用字段辅助函数，避免手写 "error", err 这类字符串。

// slogAttr 构造一个 slog.Attr，可作为任意日志函数的 args 传入。
func slogAttr(key string, value any) slog.Attr {
	return slog.Attr{Key: key, Value: slog.AnyValue(value)}
}

// Err 返回 "error", err 字段。
func Err(err error) any {
	return slogAttr("error", err)
}

// UserID 返回 "user_id", id 字段。
func UserID(id string) any {
	return slogAttr("user_id", id)
}

// UserUUID 返回 "user_uuid", uuid 字段。
func UserUUID(uuid string) any {
	return slogAttr("user_uuid", uuid)
}

// Module 返回 "module", name 字段。
func Module(name string) any {
	return slogAttr("module", name)
}

