// Package handler 提供发音评测相关的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/service/evaluation"
)

// EvaluationHandler 发音评测处理器
type EvaluationHandler struct {
	evaluationService evaluation.Service
}

// NewEvaluationHandler 创建发音评测处理器
func NewEvaluationHandler(evaluationService evaluation.Service) *EvaluationHandler {
	return &EvaluationHandler{
		evaluationService: evaluationService,
	}
}

// SubmitEvaluation 提交发音评测
// POST /api/v1/evaluate
func (h *EvaluationHandler) SubmitEvaluation(c *gin.Context) {
	// TODO: 解析请求参数
	// TODO: 调用评测服务
	// TODO: 返回评测结果

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// GetEvaluationResult 获取评测结果
// GET /api/v1/evaluate/:id
func (h *EvaluationHandler) GetEvaluationResult(c *gin.Context) {
	id := c.Param("id")

	// TODO: 调用服务获取评测结果
	_ = id

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// GetEvaluationHistory 获取评测历史
// GET /api/v1/evaluate/history
func (h *EvaluationHandler) GetEvaluationHistory(c *gin.Context) {
	// TODO: 获取用户 ID
	// TODO: 调用服务获取历史记录

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}
