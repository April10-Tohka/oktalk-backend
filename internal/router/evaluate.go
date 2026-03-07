// Package router 提供 AI 发音纠正路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupEvaluateRoutes 注册 AI 发音纠正路由（需认证）
func setupEvaluateRoutes(rg *gin.RouterGroup, h *handler.EvaluateHandler) {
	eval := rg.Group("/evaluate")
	{
		// 同步发音评测
		eval.POST("/MVP", h.EvaluateMVP)
		// 提交发音评测请求
		eval.POST("/submit", h.SubmitEvaluation)
		// 查询发音评测处理结果
		eval.GET("/result/:task_id", h.GetEvaluationResult)
		// 获取当前用户的评测历史列表
		eval.GET("/history", h.GetEvaluationHistory)
		// 获取某次评测的完整详情
		eval.GET("/:eval_id/detail", h.GetEvaluationDetail)
		// 删除指定发音评测记录
		eval.DELETE("/:eval_id", h.DeleteEvaluation)
		// 获取指定文本的标准发音音频
		eval.GET("/reference-audio/:text_id", h.GetReferenceAudio)
	}
}
