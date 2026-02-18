// Package router 提供 AI 语音对话路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupChatRoutes 注册 AI 语音对话路由（需认证）
// C-0 ~ C-6
func setupChatRoutes(rg *gin.RouterGroup, h *handler.ChatHandler) {
	chat := rg.Group("/chat")
	{
		chat.POST("/MVP", h.ChatMVP)                          // C-0
		chat.POST("/submit", h.SubmitChat)                    // C-1
		chat.GET("/result/:task_id", h.GetChatResult)         // C-2
		chat.GET("/history/:session_id", h.GetChatHistory)    // C-3
		chat.DELETE("/session/:session_id", h.DeleteSession)  // C-4
		chat.GET("/sessions", h.GetSessions)                  // C-5
		chat.POST("/feedback", h.SubmitChatFeedback)          // C-6
	}
}
