// Package router 提供 HTTP 路由配置
// 负责注册所有模块的路由和中间件
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/handler"
	"pronunciation-correction-system/internal/handler/middleware"
)

// Setup 初始化并返回路由引擎
func Setup(cfg *config.Config, handlers *handler.Handlers) *gin.Engine {
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由引擎
	r := gin.New()

	// 注册全局中间件（顺序重要）
	r.Use(middleware.RecoveryMiddleware())     // 1. Panic 恢复（最外层）
	r.Use(middleware.TraceMiddleware())        // 2. 生成 TraceID
	r.Use(middleware.LoggerMiddleware())       // 3. 请求日志（依赖 TraceID）
	r.Use(middleware.CORSMiddleware())         // 4. CORS 跨域
	r.Use(middleware.ErrorHandlerMiddleware()) // 5. 错误处理

	// 健康检查路由（无需认证）
	setupHealthRoutes(r)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证路由（无需认证）
		setupAuthRoutes(v1, handlers.User)

		// 需要认证的路由
		authenticated := v1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			// 用户路由
			setupUserRoutes(authenticated, handlers.User)

			// 对话路由
			setupChatRoutes(authenticated)

			// 评测路由
			setupEvaluateRoutes(authenticated, handlers.Evaluation, handlers.Feedback)

			// 报告路由
			setupReportRoutes(authenticated)

			// 资源路由
			setupResourceRoutes(authenticated)
		}
	}

	return r
}
