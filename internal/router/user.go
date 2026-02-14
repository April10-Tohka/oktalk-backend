// Package router 提供用户相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupUserRoutes 注册用户路由
// GET/PUT /api/v1/user/*
func setupUserRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	user := rg.Group("/user")
	{
		// 获取用户信息
		user.GET("/profile", userHandler.GetProfile)

		// 更新用户信息
		user.PUT("/profile", userHandler.UpdateProfile)

		// 获取学习统计
		user.GET("/stats", userHandler.GetLearningStats)
	}
}
