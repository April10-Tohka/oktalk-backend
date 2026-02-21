// Package handler 提供 AI 发音纠正 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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
	fileHeader, err := c.FormFile("audio_file")
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp missing audio_file", "error", err)
		BadRequest(c, "audio_file is required")
		return
	}
	audioType := strings.ToLower(strings.TrimSpace(c.PostForm("audio_type")))
	if audioType == "" {
		audioType = "wav"
	}
	textID := strings.TrimSpace(c.PostForm("text_id"))
	if textID == "" {
		logger.ErrorContext(c.Request.Context(), "evaluate mvp missing text_id", "error", errors.New("text_id is required"))
		BadRequest(c, "text_id is required")
		return
	}
	category := strings.TrimSpace(c.PostForm("category"))
	if category == "" {
		category = "read_sentence"
	}
	difficultyLevel := strings.TrimSpace(c.PostForm("difficulty_level"))
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
func (h *EvaluateHandler) SubmitEvaluation(c *gin.Context) {
	InternalError(c, "not implemented")
}

// GetEvaluationResult GET /api/v1/evaluate/result/:eval_id
func (h *EvaluateHandler) GetEvaluationResult(c *gin.Context) {
	InternalError(c, "not implemented")
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
