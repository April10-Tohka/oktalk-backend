// Package router 提供对话相关路由
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupChatRoutes 注册对话路由
// POST/GET /api/v1/chat/*
func setupChatRoutes(rg *gin.RouterGroup) {
	// TODO: 注入 ChatHandler 依赖

	chat := rg.Group("/chat")
	{
		// 创建新对话
		chat.POST("/session", func(c *gin.Context) {
			// TODO: 实现创建对话
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 发送消息
		chat.POST("/message", func(c *gin.Context) {
			// TODO: 实现发送消息
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取对话历史
		chat.GET("/session/:id", func(c *gin.Context) {
			// TODO: 实现获取对话历史
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 获取所有对话列表
		chat.GET("/sessions", func(c *gin.Context) {
			// TODO: 实现获取对话列表
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})

		// 删除对话
		chat.DELETE("/session/:id", func(c *gin.Context) {
			// TODO: 实现删除对话
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		})
	}
}
