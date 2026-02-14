// Package router 提供评测相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupEvaluateRoutes 注册评测路由
// POST/GET /api/v1/evaluate/*
func setupEvaluateRoutes(rg *gin.RouterGroup, evaluationHandler *handler.EvaluationHandler, feedbackHandler *handler.FeedbackHandler) {
	evaluate := rg.Group("/evaluate")
	{
		// 提交发音评测
		evaluate.POST("", evaluationHandler.SubmitEvaluation)

		// 获取评测结果
		evaluate.GET("/:id", evaluationHandler.GetEvaluationResult)

		// 获取评测历史
		evaluate.GET("/history", evaluationHandler.GetEvaluationHistory)

		// 获取评测反馈
		evaluate.GET("/:id/feedback", feedbackHandler.GetFeedback)

		// 获取示范音频
		evaluate.GET("/:id/demo-audio", feedbackHandler.GetDemoAudio)
	}
}
