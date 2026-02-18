// Package service 提供 AI 发音纠正业务逻辑
package service

import (
	"context"
	"log/slog"
)

// ===== 请求结构 =====

// EvaluateMVPRequest MVP 同步发音评测请求
type EvaluateMVPRequest struct {
	AudioData []byte
	AudioType string // wav / mp3
	TextID    string
	UserID    string
}

// SubmitEvaluationRequest 异步发音评测提交请求
type SubmitEvaluationRequest struct {
	AudioData      []byte
	AudioType      string
	TextID         string
	ReferenceText  string
	Language       string // zh_CN / en_US
	AssessmentType string // sentence / word / paragraph
	UserID         string
}

// EvalHistoryRequest 评测历史查询请求
type EvalHistoryRequest struct {
	UserID   string
	TextID   string
	DateFrom string
	DateTo   string
	Page     int
	PageSize int
	OrderBy  string // created_at / score
	Order    string // asc / desc
}

// ===== 响应结构 =====

// EvaluationResultResponse 发音评测完整结果
type EvaluationResultResponse struct {
	EvalID           string            `json:"eval_id"`
	Status           string            `json:"status"`
	TextID           string            `json:"text_id"`
	ReferenceText    string            `json:"reference_text"`
	OverallScore     float64           `json:"overall_score"`
	Scores           *EvalScores       `json:"scores"`
	DurationMs       int               `json:"duration_ms"`
	ProblemWords     []string          `json:"problem_words,omitempty"`
	DetailedFeedback *DetailedFeedback `json:"detailed_feedback"`
	ReferenceAudio   string            `json:"reference_audio"`
	CreatedAt        string            `json:"created_at"`
}

// EvalScores 评测分项得分
type EvalScores struct {
	Pronunciation float64 `json:"pronunciation"`
	Fluency       float64 `json:"fluency"`
	Integrity     float64 `json:"integrity"`
}

// DetailedFeedback 详细反馈
type DetailedFeedback struct {
	Strengths    []string `json:"strengths"`
	Improvements []string `json:"improvements"`
	Suggestions  []string `json:"suggestions"`
}

// EvalSummary 评测摘要（列表用）
type EvalSummary struct {
	EvalID        string      `json:"eval_id"`
	TextID        string      `json:"text_id"`
	ReferenceText string      `json:"reference_text"`
	OverallScore  float64     `json:"overall_score"`
	Scores        *EvalScores `json:"scores"`
	CreatedAt     string      `json:"created_at"`
	Status        string      `json:"status"`
}

// ReferenceAudioResponse 标准发音音频响应
type ReferenceAudioResponse struct {
	TextID        string `json:"text_id"`
	ReferenceText string `json:"reference_text"`
	AudioURL      string `json:"audio_url"`
	DurationMs    int    `json:"duration_ms"`
}

// ===== Service 接口 =====

// EvaluateService AI 发音纠正业务接口
type EvaluateService interface {
	// EvaluateMVP 同步发音评测 MVP（讯飞评测 → LLM 分级反馈 → TTS 合成）
	EvaluateMVP(ctx context.Context, req *EvaluateMVPRequest) ([]byte, error)

	// SubmitEvaluation 提交异步发音评测任务
	SubmitEvaluation(ctx context.Context, req *SubmitEvaluationRequest) (evalID string, err error)

	// GetEvaluationResult 查询异步评测结果
	GetEvaluationResult(ctx context.Context, evalID string) (*EvaluationResultResponse, error)

	// GetEvaluationHistory 获取用户评测历史列表
	GetEvaluationHistory(ctx context.Context, req *EvalHistoryRequest) ([]*EvalSummary, int64, error)

	// GetEvaluationDetail 获取单次评测完整详情
	GetEvaluationDetail(ctx context.Context, evalID string) (*EvaluationResultResponse, error)

	// DeleteEvaluation 删除评测记录
	DeleteEvaluation(ctx context.Context, evalID, userID string) error

	// GetReferenceAudio 获取指定文本的标准发音音频
	GetReferenceAudio(ctx context.Context, textID string) (*ReferenceAudioResponse, error)
}

// ===== 空实现 =====

// evaluateServiceImpl Evaluate Service 实现（暂为空实现）
type evaluateServiceImpl struct {
	// TODO: Step2 注入依赖
	// evaluationRepo    db.PronunciationEvaluationRepository
	// evaluationProvider domain.EvaluationProvider
	// llmProvider        domain.LLMProvider
	// ttsProvider        domain.TTSProvider
	// ossProvider        domain.OSSProvider
	logger *slog.Logger
}

// NewEvaluateService 创建 EvaluateService
func NewEvaluateService(logger *slog.Logger) EvaluateService {
	return &evaluateServiceImpl{logger: logger}
}

func (s *evaluateServiceImpl) EvaluateMVP(ctx context.Context, req *EvaluateMVPRequest) ([]byte, error) {
	// TODO: Step2 实现
	// 1. 讯飞语音评测 API 评分
	// 2. 根据分数计算 S/A/B/C 级别
	// 3. LLM 生成分级反馈文本
	// 4. TTS 合成反馈音频
	// 5. 保存评测记录到数据库
	// 6. 返回音频二进制数据
	return nil, nil
}

func (s *evaluateServiceImpl) SubmitEvaluation(ctx context.Context, req *SubmitEvaluationRequest) (string, error) {
	// TODO: Step3 实现异步任务
	// 1. 生成 eval_id
	// 2. 创建异步任务（讯飞评测 → LLM 反馈 → TTS 合成）
	// 3. 将任务提交到队列
	// 4. 返回 eval_id
	return "", nil
}

func (s *evaluateServiceImpl) GetEvaluationResult(ctx context.Context, evalID string) (*EvaluationResultResponse, error) {
	// TODO: Step3 实现
	// 1. 从缓存/数据库查询任务状态
	// 2. 如果完成，返回完整评测结果
	// 3. 如果处理中，返回进度信息
	// 4. 如果失败，返回错误信息
	return nil, nil
}

func (s *evaluateServiceImpl) GetEvaluationHistory(ctx context.Context, req *EvalHistoryRequest) ([]*EvalSummary, int64, error) {
	// TODO: Step2 实现
	// 1. 查询 pronunciation_evaluations 表
	// 2. 按 order_by + order 排序
	// 3. 支持 text_id / date_from / date_to 过滤
	// 4. 分页返回评测摘要
	return nil, 0, nil
}

func (s *evaluateServiceImpl) GetEvaluationDetail(ctx context.Context, evalID string) (*EvaluationResultResponse, error) {
	// TODO: Step2 实现
	// 1. 查询 pronunciation_evaluations 表
	// 2. 解析 JSON 字段（phonemes, detailed_feedback）
	// 3. 返回完整评测详情
	return nil, nil
}

func (s *evaluateServiceImpl) DeleteEvaluation(ctx context.Context, evalID, userID string) error {
	// TODO: Step2 实现
	// 1. 验证用户对该评测记录的所有权
	// 2. 软删除评测记录
	return nil
}

func (s *evaluateServiceImpl) GetReferenceAudio(ctx context.Context, textID string) (*ReferenceAudioResponse, error) {
	// TODO: Step2 实现
	// 1. 查询文本资源获取标准文本
	// 2. 查询或生成标准发音音频（TTS）
	// 3. 返回音频 URL
	return nil, nil
}
