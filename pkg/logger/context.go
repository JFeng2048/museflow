package logger

import "context"

// context 键类型，避免与其他包字符串键冲突。
type ctxKey string

const (
	traceIDKey   ctxKey = "trace_id"
	requestIDKey ctxKey = "request_id"
)

// WithTraceID 将 trace_id 注入 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithRequestID 将 request_id 注入 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// TraceIDFromContext 从 context 取出 trace_id，无则返回空串。
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestIDFromContext 从 context 取出 request_id，无则返回空串。
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
