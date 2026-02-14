// Package router 提供报告相关路由
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupReportRoutes 注册报告路由
// POST/GET /api/v1/report/*
func setupReportRoutes(rg *gin.RouterGroup) {
	// TODO: 注入 ReportHandler 依赖

	report := rg.Group("/report")
	{
		// 生成学习报告
		report.POST("/generate", func(c *gin.Context) {
			// TODO: 实现生成报告
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取报告详情
		report.GET("/:id", func(c *gin.Context) {
			// TODO: 实现获取报告
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取报告列表
		report.GET("/list", func(c *gin.Context) {
			// TODO: 实现获取报告列表
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取最新周报
		report.GET("/weekly/latest", func(c *gin.Context) {
			// TODO: 实现获取最新周报
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})
	}
}
