// Package router 提供认证相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupAuthRoutes 注册认证路由
// POST /api/v1/auth/*
func setupAuthRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	auth := rg.Group("/auth")
	{
		// 用户注册
		auth.POST("/register", userHandler.Register)

		// 用户登录
		auth.POST("/login", userHandler.Login)

		// 用户登出
		auth.POST("/logout", userHandler.Logout)
	}
}
