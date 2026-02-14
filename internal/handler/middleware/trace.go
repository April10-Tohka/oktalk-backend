// Package middleware 提供 HTTP 中间件
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"pronunciation-correction-system/internal/pkg/logger"
)

// TraceMiddleware TraceID 中间件
// 为每个 HTTP 请求生成唯一的 TraceID，并注入到 context 中
//
// 工作流程：
//  1. 检查请求头 X-Trace-ID（或 X-Request-ID），支持链路追踪透传
//  2. 如果没有，生成新的 UUID 作为 TraceID
//  3. 将 TraceID 存入 gin.Context（方便 Handler 获取）
//  4. 将 TraceID 注入 Go context（方便 Service/Repository 层通过 ctx 获取）
//  5. 将 TraceID 写入响应头 X-Trace-ID
//
// 使用方式：
//
//	router.Use(middleware.TraceMiddleware())
//
// 后续代码获取 TraceID：
//
//	// 在 Handler 中通过 gin.Context
//	traceID := c.GetString("trace_id")
//
//	// 在 Service 中通过 Go context
//	traceID := logger.TraceIDFromContext(ctx)
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 尝试从请求头获取 TraceID（支持链路追踪透传）
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetHeader("X-Request-ID")
		}

		// 2. 如果没有，生成新的 UUID
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 3. 存入 gin.Context（方便 Handler 层获取）
		c.Set("trace_id", traceID)

		// 4. 注入 Go context（方便 Service/Repository 层通过 ctx 获取）
		ctx := logger.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// 5. 设置响应头（方便客户端追踪）
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
