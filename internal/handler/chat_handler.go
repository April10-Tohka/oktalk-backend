// Package handler 提供 AI 语音对话 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/handler/middleware"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/service"
)

// ===================== WebSocket Upgrader =====================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16 * 1024, // 16KB，适合 PCM 音频块
	WriteBufferSize: 16 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发阶段允许所有来源，生产环境应限制
	},
}

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
	type chatMVPForm struct {
		ConversationID   string                `form:"conversation_id" binding:"required"`
		AudioType        string                `form:"audio_type" binding:"required"`
		ConversationType string                `form:"conversation_type"`
		DifficultyLevel  string                `form:"difficulty_level"`
		AudioFile        *multipart.FileHeader `form:"audio_file" binding:"required"`
	}
	var form chatMVPForm
	if err := c.ShouldBind(&form); err != nil {
		logger.ErrorContext(c.Request.Context(), "chat mvp bind form failed", "error", err)
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	conversationID := form.ConversationID
	fileHeader := form.AudioFile
	audioType := form.AudioType
	conversationType := form.ConversationType
	difficultyLevel := form.DifficultyLevel

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

	// 步骤 3：从 gin.Context 获取 user_id
	// 从 gin.Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
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
		UserID:           userID.(string),
		ConversationID:   conversationID,
	})
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp service failed", "error", err)
		InternalError(c, err.Error())
		return
	}

	// 步骤 6：返回音频流
	c.Data(http.StatusOK, "audio/mpeg", audioReply)
}

// StartSession POST /api/v1/chat/start-session
// 创建新的对话会话
func (h *ChatHandler) StartSession(c *gin.Context) {

	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	result, err := h.chatService.StartSession(c.Request.Context(), &service.StartSessionRequest{
		UserID: userID.(string),
	})
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	OK(c, result)
}

// SubmitChat POST /api/v1/chat/submit
// 提交异步语音对话请求，返回 task_id
func (h *ChatHandler) SubmitChat(c *gin.Context) {
	// 步骤 1：解析 multipart/form-data
	type submitChatForm struct {
		ConversationID   string                `form:"conversation_id" binding:"required"`
		AudioType        string                `form:"audio_type"`
		ConversationType string                `form:"conversation_type"`
		DifficultyLevel  string                `form:"difficulty_level"`
		Topic            string                `form:"topic"`
		AudioFile        *multipart.FileHeader `form:"audio_file" binding:"required"`
	}
	var form submitChatForm
	if err := c.ShouldBind(&form); err != nil {
		logger.ErrorContext(c.Request.Context(), "submit chat bind form failed", "error", err)
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 步骤 2：检查文件大小 ≤ 10MB
	if form.AudioFile.Size > 10*1024*1024 {
		BadRequest(c, "audio file too large, max 10MB")
		return
	}

	// 步骤 3：读取音频数据
	file, err := form.AudioFile.Open()
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit chat open file failed", "error", err)
		InternalError(c, "failed to read audio file")
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit chat read file failed", "error", err)
		InternalError(c, "failed to read audio data")
		return
	}

	// 步骤 4：从 Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	// 步骤 5：调用 Service
	taskID, err := h.chatService.SubmitChat(c.Request.Context(), &service.SubmitChatRequest{
		AudioData:        audioData,
		AudioType:        form.AudioType,
		ConversationID:   form.ConversationID,
		ConversationType: form.ConversationType,
		DifficultyLevel:  form.DifficultyLevel,
		Topic:            form.Topic,
		UserID:           userID.(string),
	})
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit chat service failed", "error", err)
		var rlErr *domain.RateLimitError
		if errors.As(err, &rlErr) {
			TooManyRequests(c, rlErr.RetryAfterSec, fmt.Sprintf("请求过于频繁，请 %d 秒后重试", rlErr.RetryAfterSec))
			return
		}
		InternalError(c, err.Error())
		return
	}

	// 步骤 6：返回成功响应
	OK(c, gin.H{
		"task_id":         taskID,
		"conversation_id": form.ConversationID,
		"status":          "pending",
		"message":         "语音对话任务已提交，请轮询查询结果",
	})
}

// GetChatResult GET /api/v1/chat/result/:task_id
// 查询异步语音对话处理结果
func (h *ChatHandler) GetChatResult(c *gin.Context) {
	// 步骤 1：解析路径参数
	taskID := c.Param("task_id")
	if taskID == "" {
		BadRequest(c, "task_id is required")
		return
	}

	// 步骤 2：调用 Service
	result, err := h.chatService.GetChatResult(c.Request.Context(), taskID)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "get chat result failed", "task_id", taskID, "error", err)
		InternalError(c, err.Error())
		return
	}

	// 步骤 3：返回结果
	OK(c, result)
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

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	// 1. 获取认证信息（由 Auth 中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("[FreeTalk] user_id missing in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		slog.Error("[FreeTalk] user_id invalid")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息无效",
		})
		return
	}
	// 2. 解析query获取conversation_id
	type freetalkRequestBody struct {
		ConversationID string `form:"conversation_id" binding:"required"`
	}
	var reqBody freetalkRequestBody
	if err := c.ShouldBindQuery(&reqBody); err != nil {
		slog.Error("[FreeTalk] bind query failed", "error", err)
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	conversationID := reqBody.ConversationID
	if conversationID == "" {
		slog.Error("[FreeTalk] missing conversation_id")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 conversation_id 参数",
		})
		return
	}
	slog.Info("[FreeTalk] request",
		"user_id", userIDStr,
		"conversation_id", conversationID,
	)
	// 3. 验证 free talk 模式语音对话请求
	err := h.chatService.ValidateFreetalk(c.Request.Context(), &service.ValidateFreetalkRequest{
		ConversationID: conversationID,
		UserID:         userIDStr,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "验证 free talk 模式语音对话请求失败",
		})
		return
	}
	// 4. 升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("[FreeTalk] WebSocket upgrade failed",
			"error", err,
			"user_id", userIDStr,
			"conversation_id", conversationID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "WebSocket upgrade failed: " + err.Error(),
		})
		return
	}

	slog.Info("[FreeTalk] WebSocket connected",
		"user_id", userIDStr,
		"conversation_id", conversationID,
	)

	// 5. 处理websocket
	h.chatService.HandleFreetalk(c.Request.Context(), &service.HandleFreetalkRequest{
		Conn:           conn,
		UserID:         userIDStr,
		ConversationID: conversationID,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":            200,
		"message":         "free talk 模式语音对话已提交成功",
		"conversation_id": conversationID,
	})
}
