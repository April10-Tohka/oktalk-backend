// Package router 提供用户相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupUserRoutes 注册用户路由（需认证）
// U-1 ~ U-2
func setupUserRoutes(rg *gin.RouterGroup, h *handler.UserHandler) {
	user := rg.Group("/user")
	{
		user.GET("/profile", h.GetProfile)    // U-1
		user.PUT("/profile", h.UpdateProfile) // U-2
	}
}
