// Package handler 提供 AI 发音纠正 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"

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
// 同步发音评测 MVP（讯飞评测 + 分级反馈 + TTS，返回音频流）
func (h *EvaluateHandler) EvaluateMVP(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 multipart/form-data: audio_file, audio_type, text_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.evaluateService.EvaluateMVP(ctx, req)
	// 4. 成功：c.Data(200, "audio/mpeg", audioData)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// SubmitEvaluation POST /api/v1/evaluate/submit
// 提交异步发音评测请求，返回 eval_id
func (h *EvaluateHandler) SubmitEvaluation(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析 multipart/form-data: audio_file, audio_type, text_id, reference_text, language, assessment_type
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.evaluateService.SubmitEvaluation(ctx, req)
	// 4. 成功：OK(c, gin.H{"eval_id": evalID, "text_id": req.TextID, "status": "pending"})
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetEvaluationResult GET /api/v1/evaluate/result/:eval_id
// 查询异步评测结果
func (h *EvaluateHandler) GetEvaluationResult(c *gin.Context) {
	// TODO: Step3 实现
	// 1. 解析路径参数: eval_id
	// 2. 调用 h.evaluateService.GetEvaluationResult(ctx, evalID)
	// 3. 成功：OK(c, result)
	// 4. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetEvaluationHistory GET /api/v1/evaluate/history
// 获取用户评测历史列表
func (h *EvaluateHandler) GetEvaluationHistory(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析查询参数: text_id, date_from, date_to, page(默认1), page_size(默认20), order_by(默认created_at), order(默认desc)
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.evaluateService.GetEvaluationHistory(ctx, req)
	// 4. 成功：OKPage(c, items, page, pageSize, total)
	// 5. 失败：InternalError(c, err.Error())
	InternalError(c, "not implemented")
}

// GetEvaluationDetail GET /api/v1/evaluate/:eval_id/detail
// 获取单次评测完整详情（含音素分析）
func (h *EvaluateHandler) GetEvaluationDetail(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: eval_id
	// 2. 调用 h.evaluateService.GetEvaluationDetail(ctx, evalID)
	// 3. 成功：OK(c, result)
	// 4. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}

// DeleteEvaluation DELETE /api/v1/evaluate/:eval_id
// 删除评测记录
func (h *EvaluateHandler) DeleteEvaluation(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: eval_id
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.evaluateService.DeleteEvaluation(ctx, evalID, userID)
	// 4. 成功：OK(c, gin.H{"eval_id": evalID, "message": "评测记录已删除"})
	// 5. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}

// GetReferenceAudio GET /api/v1/evaluate/reference-audio/:text_id
// 获取指定文本的标准发音音频
func (h *EvaluateHandler) GetReferenceAudio(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析路径参数: text_id
	// 2. 调用 h.evaluateService.GetReferenceAudio(ctx, textID)
	// 3. 成功：OK(c, result)
	// 4. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}
