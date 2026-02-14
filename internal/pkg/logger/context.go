// Package logger 提供基于 log/slog 的结构化日志功能
package logger

import "context"

// contextKey 自定义 context key 类型，避免与其他包冲突
type contextKey string

const (
	// traceIDKey 在 context 中存储 TraceID 的 key
	traceIDKey contextKey = "trace_id"
)

// WithTraceID 将 TraceID 注入到 context 中
// 典型用法：在 HTTP 中间件中为每个请求生成唯一 TraceID 并注入
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext 从 context 中提取 TraceID
// 如果 context 中没有 TraceID，返回空字符串
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}
