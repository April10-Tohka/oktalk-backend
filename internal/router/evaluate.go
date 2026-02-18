// Package router 提供 AI 发音纠正路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupEvaluateRoutes 注册 AI 发音纠正路由（需认证）
// E-0 ~ E-6
func setupEvaluateRoutes(rg *gin.RouterGroup, h *handler.EvaluateHandler) {
	eval := rg.Group("/evaluate")
	{
		// ── 静态路径（优先匹配）──
		eval.POST("/MVP", h.EvaluateMVP)                          // E-0
		eval.POST("/submit", h.SubmitEvaluation)                  // E-1
		eval.GET("/history", h.GetEvaluationHistory)              // E-3
		eval.GET("/reference-audio/:text_id", h.GetReferenceAudio) // E-6

		// ── 参数路径 ──
		eval.GET("/result/:eval_id", h.GetEvaluationResult)       // E-2
		eval.GET("/:eval_id/detail", h.GetEvaluationDetail)       // E-4
		eval.DELETE("/:eval_id", h.DeleteEvaluation)              // E-5
	}
}
