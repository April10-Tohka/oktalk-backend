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
		// 固定路径须先于 /:report_id 注册，避免被通配吞掉
		report.POST("/MVP", h.ReportMVP)
		report.POST("/submit", h.GenerateReport)
		report.POST("/generate", h.GenerateWeeklyReport)
		report.GET("/list", h.GetReportList)
		report.GET("/dashboard", h.GetDashboard)
		report.GET("/result/:task_id", h.GetReport)
		report.GET("/:report_id/status", h.GetReportStatus)
		report.DELETE("/:report_id", h.DeleteReport)
		// 周报详情（主键）；放在最后
		report.GET("/:report_id", h.GetWeeklyReportDetail)
	}
}
