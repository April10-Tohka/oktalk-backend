// Package router 提供智能学习报告路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupReportRoutes 注册智能学习报告路由（需认证）
// R-0 ~ R-6
func setupReportRoutes(rg *gin.RouterGroup, h *handler.ReportHandler) {
	report := rg.Group("/report")
	{
		// ── 静态路径（优先匹配）──
		report.POST("/MVP", h.ReportMVP)              // R-0
		report.POST("/generate", h.GenerateReport)    // R-1
		report.GET("/list", h.GetReportList)           // R-4
		report.GET("/dashboard", h.GetDashboard)       // R-6

		// ── 参数路径 ──
		report.GET("/:report_id/status", h.GetReportStatus) // R-2
		report.GET("/:report_id", h.GetReport)              // R-3
		report.DELETE("/:report_id", h.DeleteReport)        // R-5
	}
}
