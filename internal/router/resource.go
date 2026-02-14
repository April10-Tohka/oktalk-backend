// Package router 提供资源相关路由
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupResourceRoutes 注册资源路由
// GET /api/v1/resources/*
func setupResourceRoutes(rg *gin.RouterGroup) {
	// TODO: 注入 ResourceHandler 依赖

	resources := rg.Group("/resources")
	{
		// 获取文本资源列表
		resources.GET("/texts", func(c *gin.Context) {
			// TODO: 实现获取文本资源
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取单个文本详情
		resources.GET("/texts/:id", func(c *gin.Context) {
			// TODO: 实现获取文本详情
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取场景列表
		resources.GET("/scenarios", func(c *gin.Context) {
			// TODO: 实现获取场景列表
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取难度等级
		resources.GET("/levels", func(c *gin.Context) {
			// TODO: 实现获取难度等级
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})
	}
}

// setupHealthRoutes 注册健康检查路由
func setupHealthRoutes(r *gin.Engine) {
	health := r.Group("")
	{
		health.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		health.GET("/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		health.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
	}
}
