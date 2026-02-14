// Package middleware 提供请求日志中间件
package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/pkg/logger"
)

// LoggerMiddleware 请求日志中间件
// 记录每个 HTTP 请求的关键信息：方法、路径、状态码、耗时等
//
// 注意：此中间件依赖 TraceMiddleware，需要在其之后注册
//
// 输出示例（控制台）：
//
//	time=2024-01-15 10:30:45 level=INFO msg="request completed"
//	  trace_id=abc-123 method=POST path=/api/v1/evaluate
//	  status=200 latency=128ms
//
// 使用方式：
//
//	router.Use(middleware.TraceMiddleware())   // 先注册 Trace
//	router.Use(middleware.LoggerMiddleware())  // 再注册 Logger
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取带有 TraceID 的 context
		ctx := c.Request.Context()

		// 记录请求开始时间
		startTime := time.Now()

		// 请求信息
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 记录请求开始
		slog.InfoContext(ctx, "request started",
			"method", method,
			"path", fullPath(path, query),
			"client_ip", clientIP,
		)

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		// 根据状态码选择日志级别
		attrs := []any{
			"method", method,
			"path", fullPath(path, query),
			"status", statusCode,
			"latency", formatLatency(latency),
			"client_ip", clientIP,
			"user_agent", userAgent,
			"body_size", c.Writer.Size(),
		}

		// 如果有错误信息，附加到日志
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case statusCode >= 500:
			// 服务端错误 → Error 级别（自动附加调用栈）
			slog.ErrorContext(ctx, "request completed with server error", attrs...)
		case statusCode >= 400:
			// 客户端错误 → Warn 级别
			slog.WarnContext(ctx, "request completed with client error", attrs...)
		default:
			// 正常请求 → Info 级别
			slog.InfoContext(ctx, "request completed", attrs...)
		}

		// 慢请求警告（超过 3 秒）
		if latency > 3*time.Second {
			logger.WarnContext(ctx, "slow request detected",
				"method", method,
				"path", path,
				"latency", formatLatency(latency),
			)
		}
	}
}

// fullPath 拼接路径和查询参数
func fullPath(path, query string) string {
	if query != "" {
		return path + "?" + query
	}
	return path
}

// formatLatency 格式化耗时为可读字符串
func formatLatency(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	case d < time.Second:
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}
