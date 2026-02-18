// Package router 提供健康检查、系统状态、学习资源路由
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupHealthRoutes 注册健康检查路由（无需认证）
func setupHealthRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
}

// setupSystemRoutes 注册系统路由（无需认证）
// S-1
func setupSystemRoutes(rg *gin.RouterGroup, h *handler.SystemHandler) {
	system := rg.Group("/system")
	{
		system.GET("/status", h.GetSystemStatus) // S-1
	}
}

// setupResourceRoutes 注册学习资源路由（需认证）
// S-2
func setupResourceRoutes(rg *gin.RouterGroup, h *handler.SystemHandler) {
	resources := rg.Group("/resources")
	{
		resources.GET("/texts", h.GetLearningTexts) // S-2
	}
}
