// Package router 提供认证相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupAuthRoutes 注册认证路由（无需登录）
// A-1 ~ A-4
func setupAuthRoutes(rg *gin.RouterGroup, h *handler.AuthHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)          // A-1
		auth.POST("/register", h.Register)    // A-2
		auth.POST("/logout", h.Logout)        // A-3
		auth.POST("/refresh", h.RefreshToken) // A-4
	}
}
