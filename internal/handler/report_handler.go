// Package handler 提供智能学习报告 HTTP 处理器
package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/handler/middleware"
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
	// 1. 解析请求体
	var reqBody struct {
		ReportType string `json:"report_type" binding:"required"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 2.从 Gin Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}
	req := service.ReportMVPRequest{
		ReportType: reqBody.ReportType,
		StartDate:  reqBody.StartDate,
		EndDate:    reqBody.EndDate,
		UserID:     userID.(string),
	}

	// 3. 调用 Service
	result, err := h.reportService.ReportMVP(c.Request.Context(), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, result)
}

// GenerateReport POST /api/v1/report/generate
// 提交异步报告生成任务，返回 task_id
func (h *ReportHandler) GenerateReport(c *gin.Context) {
	// 步骤 1：解析 JSON 请求体
	var reqBody struct {
		ReportType string `json:"report_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 步骤 2：校验 report_type
	if reqBody.ReportType != "weekly" && reqBody.ReportType != "monthly" {
		BadRequest(c, "invalid report_type: must be weekly or monthly")
		return
	}

	// 步骤 3：从 Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	// 步骤 4：调用 Service
	taskID, err := h.reportService.GenerateReport(c.Request.Context(), &service.GenerateReportRequest{
		ReportType: reqBody.ReportType,
		UserID:     userID.(string),
	})
	if err != nil {
		var rlErr *domain.RateLimitError
		if errors.As(err, &rlErr) {
			TooManyRequests(c, rlErr.RetryAfterSec, fmt.Sprintf("请求过于频繁，请 %d 秒后重试", rlErr.RetryAfterSec))
			return
		}
		InternalError(c, err.Error())
		return
	}
	OK(c, gin.H{
		"task_id":     taskID,
		"report_type": reqBody.ReportType,
		"status":      "pending",
		"message":     "报告生成任务已提交，请稍后查询结果",
	})
}

// GetReportStatus GET /api/v1/report/:report_id/status
// 查询报告生成进度
func (h *ReportHandler) GetReportStatus(c *gin.Context) {
	reportID := c.Param("report_id")
	if reportID == "" {
		BadRequest(c, "report_id is required")
		return
	}

	result, err := h.reportService.GetReportStatus(c.Request.Context(), reportID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, result)
}

// GetReport GET /api/v1/report/:task_id
// 获取报告完整详情（也用于轮询异步报告结果）
func (h *ReportHandler) GetReport(c *gin.Context) {
	reportID := c.Param("task_id")
	if reportID == "" {
		BadRequest(c, "task_id is required")
		return
	}

	// 从 Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	result, err := h.reportService.GetReport(c.Request.Context(), reportID, userID.(string))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, result)
}

// GetReportList GET /api/v1/report/list
// 获取用户报告列表
func (h *ReportHandler) GetReportList(c *gin.Context) {
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "0"))
	items, _, err := h.reportService.GetReportList(c.Request.Context(), userID.(string), page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	OK(c, gin.H{"reports": items})
}

// GetReportDetail GET /api/v1/report/:report_id
// 从数据库 content 返回完整周报 JSON
func (h *ReportHandler) GetReportDetail(c *gin.Context) {
	reportID := c.Param("report_id")
	if reportID == "" {
		BadRequest(c, "report_id is required")
		return
	}
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}
	detail, err := h.reportService.GetReportDetail(c.Request.Context(), reportID, userID.(string))
	if err != nil {
		if errors.Is(err, service.ErrReportNotFound) {
			NotFound(c, "report not found")
			return
		}
		if errors.Is(err, service.ErrReportAccessDenied) {
			Forbidden(c)
			return
		}
		InternalError(c, err.Error())
		return
	}
	OK(c, detail)
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
