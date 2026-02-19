// Package handler 提供 AI 语音对话 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	httpx "pronunciation-correction-system/internal/pkg/http"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/service"
)

// ChatHandler AI 语音对话处理器
type ChatHandler struct {
	chatService service.ChatService
}

// NewChatHandler 创建 ChatHandler
func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// ChatMVP POST /api/v1/chat/MVP
// 同步语音对话 MVP（ASR + LLM + TTS，返回音频流）
func (h *ChatHandler) ChatMVP(c *gin.Context) {
	// 步骤 1：解析 multipart/form-data
	fileHeader, err := c.FormFile("audio_file")
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "chat mvp missing audio_file", "error", err)
		BadRequest(c, "audio_file is required")
		return
	}
	audioType := c.PostForm("audio_type")
	if audioType == "" {
		logger.ErrorContext(c.Request.Context(), "chat mvp missing audio_type", "error", errors.New("audio_type is required"))
		BadRequest(c, "audio_type is required")
		return
	}
	conversationType := c.PostForm("conversation_type")
	difficultyLevel := c.PostForm("difficulty_level")

	// 步骤 2：读取音频数据
	file, err := fileHeader.Open()
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "chat mvp open file failed", "error", err)
		InternalError(c, "failed to read audio file")
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "chat mvp read file failed", "error", err)
		InternalError(c, "failed to read audio data")
		return
	}

	// 步骤 3：获取 user_id
	userID := httpx.GetUserID(c)
	if userID == "" {
		logger.ErrorContext(c.Request.Context(), "chat mvp user id missing", "error", errors.New("user id is empty"))
		Unauthorized(c)
		return
	}

	// 步骤 4：设置超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 步骤 5：调用 Service
	audioReply, err := h.chatService.ChatMVP(ctx, &service.ChatMVPRequest{
		AudioData:        audioData,
		AudioType:        audioType,
		ConversationType: conversationType,
		DifficultyLevel:  difficultyLevel,
		UserID:           userID,
	})
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp service failed", "error", err)
		InternalError(c, err.Error())
		return
	}

	// 步骤 6：返回音频流
	c.Data(http.StatusOK, "audio/mpeg", audioReply)
}

// SubmitChat POST /api/v1/chat/submit
// 提交异步语音对话请求，返回 task_id
func (h *ChatHandler) SubmitChat(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析 multipart/form-data: audio_file, audio_type, session_id, user_language, topic_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.chatService.SubmitChat(ctx, req)
	// 4. 成功：OK(c, gin.H{"task_id": taskID, "session_id": req.SessionID, "status": "pending"})
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetChatResult GET /api/v1/chat/result/:task_id
// 查询异步语音对话处理结果
func (h *ChatHandler) GetChatResult(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析路径参数: task_id
	// 2. 调用 h.chatService.GetChatResult(ctx, taskID)
	// 3. 成功：OK(c, result)
	// 4. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetChatHistory GET /api/v1/chat/history/:session_id
// 获取指定会话的对话历史
func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: session_id
	// 2. 解析查询参数: page(默认1), page_size(默认20), order(默认asc)
	// 3. 从 Context 获取 user_id
	// 4. 调用 h.chatService.GetChatHistory(ctx, req)
	// 5. 成功：OKPage(c, items, page, pageSize, total)
	// 6. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// DeleteSession DELETE /api/v1/chat/session/:session_id
// 删除对话会话及其所有消息
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: session_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.chatService.DeleteSession(ctx, sessionID, userID)
	// 4. 成功：OK(c, gin.H{"session_id": sessionID, "deleted_records": count})
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetSessions GET /api/v1/chat/sessions
// 获取当前用户的所有会话列表
func (h *ChatHandler) GetSessions(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析查询参数: page(默认1), page_size(默认20)
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.chatService.GetSessions(ctx, userID, page, pageSize)
	// 4. 成功：OKPage(c, items, page, pageSize, total)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// SubmitChatFeedback POST /api/v1/chat/feedback
// 提交对话反馈
func (h *ChatHandler) SubmitChatFeedback(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 JSON 请求体: task_id, session_id, turn, rating, comment, helpful
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.chatService.SubmitChatFeedback(ctx, req)
	// 4. 成功：OK(c, gin.H{"message": "感谢您的反馈"})
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}
