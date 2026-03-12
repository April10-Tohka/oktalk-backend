// Package router 提供 AI 语音对话路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
	"pronunciation-correction-system/internal/handler/freetalk"
)

// setupChatRoutes 注册 AI 语音对话路由（需认证）
func setupChatRoutes(rg *gin.RouterGroup, h *handler.ChatHandler) {
	chat := rg.Group("/chat")
	{
		// 同步语音对话
		chat.POST("/MVP", h.ChatMVP)
		// 启动新会话
		chat.POST("/session/start", h.StartSession)
		// 提交语音对话请求
		chat.POST("/submit", h.SubmitChat)
		// 查询语音对话处理结果
		chat.GET("/result/:task_id", h.GetChatResult)
		//获取指定会话的对话历史记录
		chat.GET("/history/:session_id", h.GetChatHistory)
		// 删除指定会话及其所有对话记录
		chat.DELETE("/session/:session_id", h.DeleteSession)
		// 获取当前用户的所有会话列表
		chat.GET("/sessions", h.GetSessions)
		chat.POST("/feedback", h.SubmitChatFeedback)

		// free talk 模式的语音对话
		chat.POST("/freetalk", h.HandleWebSocket)
	}
}

// setupFreeTalkRoutes 注册 Free Talk WebSocket 路由（需认证）
func setupFreeTalkRoutes(rg *gin.RouterGroup, h *freetalk.Handler) {
	rg.GET("/chat/freetalk", h.HandleWebSocket)
}
