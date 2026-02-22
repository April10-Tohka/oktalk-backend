// Package router 提供智能学习报告路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupReportRoutes 注册智能学习报告路由（需认证）
func setupReportRoutes(rg *gin.RouterGroup, h *handler.ReportHandler) {
	report := rg.Group("/report")
	{
		// 同步生成学习报告
		report.POST("/MVP", h.ReportMVP)
		// 生成学习报告
		report.POST("/generate", h.GenerateReport)
		// 获取当前用户的所有报告列表
		report.GET("/list", h.GetReportList)
		// 获取学习总体统计数据
		report.GET("/dashboard", h.GetDashboard)
		// 查询报告生成进度
		report.GET("/:report_id/status", h.GetReportStatus)
		// 获取报告详情
		report.GET("/:report_id", h.GetReport)
		// 删除指定学习报告
		report.DELETE("/:report_id", h.DeleteReport)
	}
}
