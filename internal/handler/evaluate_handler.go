// Package handler 提供 AI 发音纠正 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/handler/middleware"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/service"
)

// EvaluateHandler AI 发音纠正处理器
type EvaluateHandler struct {
	evaluateService service.EvaluateService
}

// NewEvaluateHandler 创建 EvaluateHandler
func NewEvaluateHandler(evaluateService service.EvaluateService) *EvaluateHandler {
	return &EvaluateHandler{evaluateService: evaluateService}
}

// EvaluateMVP POST /api/v1/evaluate/MVP
// 同步发音评测 MVP（讯飞评测 → LLM 分级反馈 → TTS 合成）
func (h *EvaluateHandler) EvaluateMVP(c *gin.Context) {
	// 步骤 1：解析 multipart/form-data
	type evaluateMVPForm struct {
		AudioFile       *multipart.FileHeader `form:"audio_file" binding:"required"`
		AudioType       string                `form:"audio_type"`
		TextID          string                `form:"text_id" binding:"required"`
		Category        string                `form:"category"`
		DifficultyLevel string                `form:"difficulty_level"`
	}
	var form evaluateMVPForm
	if err := c.ShouldBind(&form); err != nil {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp bind form failed", "error", err)
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	fileHeader := form.AudioFile
	audioType := strings.ToLower(strings.TrimSpace(form.AudioType))
	if audioType == "" {
		audioType = "wav"
	}
	textID := strings.TrimSpace(form.TextID)
	category := strings.TrimSpace(form.Category)
	if category == "" {
		category = "read_sentence"
	}
	difficultyLevel := strings.TrimSpace(form.DifficultyLevel)
	if difficultyLevel == "" {
		difficultyLevel = "beginner"
	}

	// 步骤 2：读取音频数据
	file, err := fileHeader.Open()
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp open file failed", "error", err)
		InternalError(c, "failed to read audio file")
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp read file failed", "error", err)
		InternalError(c, "failed to read audio data")
		return
	}

	// 步骤 3：WAV 格式去掉 44 字节 header（讯飞评测需 PCM 裸数据）
	if audioType == "wav" && len(audioData) > 44 {
		audioData = audioData[44:]
	}

	// 步骤 4：从 Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp user id missing", "error", errors.New("user id is empty"))
		Unauthorized(c)
		return
	}

	// 步骤 5：设置超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 步骤 6：调用 Service
	resp, err := h.evaluateService.EvaluateMVP(ctx, &service.EvaluateMVPRequest{
		AudioData:       audioData,
		AudioType:       audioType,
		TextID:          textID,
		Category:        category,
		DifficultyLevel: difficultyLevel,
		UserID:          userID.(string),
	})
	if err != nil {
		logger.ErrorContext(ctx, "evaluate mvp service failed", "error", err)
		InternalError(c, err.Error())
		return
	}

	// 步骤 7：返回 JSON 响应
	OK(c, resp)
}

// SubmitEvaluation POST /api/v1/evaluate/submit
// 提交异步发音评测请求，返回 eval_id
func (h *EvaluateHandler) SubmitEvaluation(c *gin.Context) {
	// 步骤 1：解析 multipart/form-data
	type submitEvalForm struct {
		AudioFile       *multipart.FileHeader `form:"audio_file" binding:"required"`
		AudioType       string                `form:"audio_type"`
		TextID          string                `form:"text_id" binding:"required"`
		Category        string                `form:"category" binding:"required"`
		DifficultyLevel string                `form:"difficulty_level"`
	}
	var form submitEvalForm
	if err := c.ShouldBind(&form); err != nil {
		logger.ErrorContext(c.Request.Context(), "submit eval bind form failed", "error", err)
		BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 步骤 2：校验 category
	category := strings.TrimSpace(form.Category)
	if category != "read_word" && category != "read_sentence" {
		BadRequest(c, "invalid category: must be read_word or read_sentence")
		return
	}

	// 步骤 3：检查文件大小 ≤ 10MB
	if form.AudioFile.Size > 10*1024*1024 {
		BadRequest(c, "audio file too large, max 10MB")
		return
	}

	// 步骤 4：读取音频数据
	file, err := form.AudioFile.Open()
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit eval open file failed", "error", err)
		InternalError(c, "failed to read audio file")
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit eval read file failed", "error", err)
		InternalError(c, "failed to read audio data")
		return
	}

	// 步骤 5：从 Context 获取 user_id
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	// 步骤 6：调用 Service
	taskID, err := h.evaluateService.SubmitEvaluation(c.Request.Context(), &service.SubmitEvaluationRequest{
		AudioData:       audioData,
		AudioType:       strings.ToLower(strings.TrimSpace(form.AudioType)),
		TextID:          strings.TrimSpace(form.TextID),
		Category:        category,
		DifficultyLevel: strings.TrimSpace(form.DifficultyLevel),
		UserID:          userID.(string),
	})
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "submit eval service failed", "error", err)
		var rlErr *domain.RateLimitError
		if errors.As(err, &rlErr) {
			TooManyRequests(c, rlErr.RetryAfterSec, fmt.Sprintf("请求过于频繁，请 %d 秒后重试", rlErr.RetryAfterSec))
			return
		}
		InternalError(c, err.Error())
		return
	}

	// 步骤 7：返回成功响应
	OK(c, gin.H{
		"task_id": taskID,
		"text_id": form.TextID,
		"status":  "pending",
		"message": "发音评测任务已提交，请轮询查询结果",
	})
}

// GetEvaluationResult GET /api/v1/evaluate/result/:task_id
// 查询异步发音评测处理结果
func (h *EvaluateHandler) GetEvaluationResult(c *gin.Context) {
	// 步骤 1：解析路径参数
	taskID := c.Param("task_id")
	if taskID == "" {
		BadRequest(c, "task_id is required")
		return
	}

	// 步骤 2：调用 Service
	result, err := h.evaluateService.GetEvaluationResult(c.Request.Context(), taskID)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "get eval result failed", "task_id", taskID, "error", err)
		InternalError(c, err.Error())
		return
	}

	// 步骤 3：返回结果
	OK(c, result)
}

// GetEvaluationHistory GET /api/v1/evaluate/history
func (h *EvaluateHandler) GetEvaluationHistory(c *gin.Context) {
	InternalError(c, "not implemented")
}

// GetEvaluationDetail GET /api/v1/evaluate/:eval_id/detail
func (h *EvaluateHandler) GetEvaluationDetail(c *gin.Context) {
	InternalError(c, "not implemented")
}

// DeleteEvaluation DELETE /api/v1/evaluate/:eval_id
func (h *EvaluateHandler) DeleteEvaluation(c *gin.Context) {
	InternalError(c, "not implemented")
}

// GetReferenceAudio GET /api/v1/evaluate/reference-audio/:text_id
func (h *EvaluateHandler) GetReferenceAudio(c *gin.Context) {
	_ = c.Param("text_id")
	InternalError(c, "not implemented")
}

// handleAudioResponse 返回音频流响应
func handleAudioResponse(c *gin.Context, audioData []byte) {
	c.Data(http.StatusOK, "audio/mpeg", audioData)
}
