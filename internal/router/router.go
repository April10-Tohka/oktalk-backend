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
// handlers: 通过依赖注入传入的所有 Handler 实例
func Setup(cfg *config.Config, handlers *handler.Handlers) *gin.Engine {
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由引擎
	r := gin.New()

	// ── 全局中间件（顺序重要）──
	r.Use(middleware.RecoveryMiddleware())     // 1. Panic 恢复（最外层）
	r.Use(middleware.TraceMiddleware())        // 2. 生成 TraceID
	r.Use(middleware.LoggerMiddleware())       // 3. 请求日志（依赖 TraceID）
	r.Use(middleware.CORSMiddleware())         // 4. CORS 跨域
	r.Use(middleware.ErrorHandlerMiddleware()) // 5. 错误处理

	// ── 公开路由（无需认证）──
	setupHealthRoutes(r)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证路由（无需登录）
		setupAuthRoutes(v1, handlers.Auth)

		// 系统状态路由（无需登录）
		setupSystemRoutes(v1, handlers.System)

		// ── 需要认证的路由 ──
		authed := v1.Group("")
		authed.Use(middleware.Auth(cfg))
		{
			setupChatRoutes(authed, handlers.Chat)         // AI 语音对话
			setupEvaluateRoutes(authed, handlers.Evaluate) // AI 发音纠正
			setupReportRoutes(authed, handlers.Report)     // 智能学习报告
			setupUserRoutes(authed, handlers.User)         // 用户信息
			setupResourceRoutes(authed, handlers.System)   // 学习资源
		}
	}

	return r
}
