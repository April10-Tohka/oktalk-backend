// Package handler 提供反馈相关的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/service/feedback"
)

// FeedbackHandler 反馈处理器
type FeedbackHandler struct {
	feedbackService feedback.Service
}

// NewFeedbackHandler 创建反馈处理器
func NewFeedbackHandler(feedbackService feedback.Service) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
	}
}

// GetFeedback 获取反馈信息
// GET /api/v1/feedback/:evaluation_id
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	evaluationID := c.Param("evaluation_id")

	// TODO: 调用服务获取反馈
	_ = evaluationID

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// GenerateFeedback 生成反馈
// POST /api/v1/feedback/generate
func (h *FeedbackHandler) GenerateFeedback(c *gin.Context) {
	// TODO: 解析请求参数
	// TODO: 调用服务生成反馈

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// GetDemoAudio 获取示范音频
// GET /api/v1/feedback/:evaluation_id/demo-audio
func (h *FeedbackHandler) GetDemoAudio(c *gin.Context) {
	evaluationID := c.Param("evaluation_id")

	// TODO: 调用服务获取示范音频 URL
	_ = evaluationID

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}
