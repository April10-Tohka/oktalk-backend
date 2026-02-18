// Package handler 提供智能学习报告 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/service"
)

// ReportHandler 智能学习报告处理器
type ReportHandler struct {
	reportService service.ReportService
}

// NewReportHandler 创建 ReportHandler
func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// ReportMVP POST /api/v1/report/MVP
// 同步生成学习报告 MVP（统计 + LLM，直接返回报告内容）
func (h *ReportHandler) ReportMVP(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 JSON 请求体: report_type, start_date, end_date
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.ReportMVP(ctx, req)
	// 4. 成功：OK(c, result)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GenerateReport POST /api/v1/report/generate
// 提交异步报告生成任务，返回 report_id
func (h *ReportHandler) GenerateReport(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析 JSON 请求体: report_type, start_date, end_date, include_evaluations, include_chat_stats, custom_prompt
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.GenerateReport(ctx, req)
	// 4. 成功：OK(c, gin.H{"report_id": reportID, "report_type": req.ReportType, "status": "generating"})
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetReportStatus GET /api/v1/report/:report_id/status
// 查询报告生成进度
func (h *ReportHandler) GetReportStatus(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析路径参数: report_id
	// 2. 调用 h.reportService.GetReportStatus(ctx, reportID)
	// 3. 成功：OK(c, result)
	// 4. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetReport GET /api/v1/report/:report_id
// 获取报告完整详情
func (h *ReportHandler) GetReport(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: report_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.GetReport(ctx, reportID, userID)
	// 4. 成功：OK(c, result)
	// 5. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}

// GetReportList GET /api/v1/report/list
// 获取用户报告列表
func (h *ReportHandler) GetReportList(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析查询参数: report_type, date_from, date_to, page(默认1), page_size(默认10), order_by, order
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.GetReportList(ctx, userID, page, pageSize)
	// 4. 成功：OKPage(c, items, page, pageSize, total)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// DeleteReport DELETE /api/v1/report/:report_id
// 删除报告
func (h *ReportHandler) DeleteReport(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: report_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.DeleteReport(ctx, reportID, userID)
	// 4. 成功：OK(c, gin.H{"report_id": reportID, "message": "报告已删除"})
	// 5. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}

// GetDashboard GET /api/v1/report/dashboard
// 获取学习统计面板
func (h *ReportHandler) GetDashboard(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析查询参数: days(默认7)
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.reportService.GetDashboard(ctx, userID)
	// 4. 成功：OK(c, result)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}
