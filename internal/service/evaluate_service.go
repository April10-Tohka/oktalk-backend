// Package service 提供 AI 发音纠正业务逻辑
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	llmPrompts "pronunciation-correction-system/internal/infrastructure/llm"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
)

// ===== 请求结构 =====

// EvaluateMVPRequest MVP 同步发音评测请求
type EvaluateMVPRequest struct {
	AudioData       []byte
	AudioType       string // wav / mp3
	TextID          string // 文本 ID（如 "text_001"）
	Category        string // read_sentence / read_word
	DifficultyLevel string // beginner / intermediate / advanced
	UserID          string
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

// EvaluateMVPResponse MVP 评测响应（对应前端布局）
type EvaluateMVPResponse struct {
	// === 顶部：综合信息 ===
	OverallScore  float64 `json:"overall_score"`  // 综合得分
	FeedbackLevel string  `json:"feedback_level"` // S / A / B / C
	LevelText     string  `json:"level_text"`     // "Perfect!" / "Good Try!" 等

	// === 分项得分 ===
	AccuracyScore  float64 `json:"accuracy_score"`  // 准确度
	FluencyScore   float64 `json:"fluency_score"`   // 流利度
	IntegrityScore float64 `json:"integrity_score"` // 完整度

	// === AI 反馈 ===
	FeedbackText     string `json:"feedback_text"`      // 反馈文本
	FeedbackAudioURL string `json:"feedback_audio_url"` // 反馈音频 URL

	// === 标准示范（可选） ===
	DemoAudio *DemoAudio `json:"demo_audio,omitempty"` // 90+ 分时为 null

	// === 单词详情 ===
	WordDetails []WordDetail `json:"word_details"` // 单词列表

	// === 其他 ===
	TargetText string `json:"target_text"` // 目标文本
	EvalID     string `json:"eval_id"`     // 评测记录 ID
}

// DemoAudio 示范音频（A/B/C 级提供）
type DemoAudio struct {
	Type     string `json:"type"`      // "word" 或 "sentence"
	Text     string `json:"text"`      // 示范内容
	AudioURL string `json:"audio_url"` // 示范音频 URL
}

// WordDetail 单词详情
type WordDetail struct {
	Word      string  `json:"word"`       // 单词
	Score     float64 `json:"score"`      // 单词得分
	IsProblem bool    `json:"is_problem"` // 是否有问题
}

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
	EvaluateMVP(ctx context.Context, req *EvaluateMVPRequest) (*EvaluateMVPResponse, error)

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

// evaluateServiceImpl Evaluate Service 实现
type evaluateServiceImpl struct {
	repos              *db.Repositories
	evaluationProvider domain.EvaluationProvider
	llmProvider        domain.LLMProvider
	ttsProvider        domain.TTSProvider
	ossProvider        domain.OSSProvider
	logger             *slog.Logger
}

// NewEvaluateService 创建 EvaluateService
func NewEvaluateService(
	repos *db.Repositories,
	evaluationProvider domain.EvaluationProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	ossProvider domain.OSSProvider,
	logger *slog.Logger,
) EvaluateService {
	return &evaluateServiceImpl{
		repos:              repos,
		evaluationProvider: evaluationProvider,
		llmProvider:        llmProvider,
		ttsProvider:        ttsProvider,
		ossProvider:        ossProvider,
		logger:             logger,
	}
}

func (s *evaluateServiceImpl) EvaluateMVP(ctx context.Context, req *EvaluateMVPRequest) (*EvaluateMVPResponse, error) {
	// ─── 基础校验 ───
	if req == nil {
		return nil, errors.New("evaluate mvp request is nil")
	}
	if len(req.AudioData) == 0 {
		return nil, errors.New("audio data is empty")
	}
	if req.UserID == "" {
		return nil, errors.New("user id is empty")
	}
	if s.evaluationProvider == nil || s.llmProvider == nil || s.ttsProvider == nil {
		return nil, errors.New("required providers not initialized")
	}
	// ─── 1. 获取目标文本（硬编码映射） ───
	targetText, ok := textIDMap[req.TextID]
	if !ok {
		return nil, fmt.Errorf("unknown text_id: %s", req.TextID)
	}
	logger.InfoContext(ctx, "evaluate mvp start", "text_id", req.TextID, "target_text", targetText)

	// ─── 2. 讯飞语音评测 ───
	evalResult, err := s.evaluationProvider.Assess(ctx, targetText, req.AudioData)
	if err != nil {
		logger.ErrorContext(ctx, "evaluate mvp assess failed", "error", err)
		return nil, fmt.Errorf("speech assessment failed: %w", err)
	}
	logger.InfoContext(ctx, "evaluate mvp assess done",
		"total_score", evalResult.TotalScore,
		"accuracy", evalResult.Accuracy,
		"fluency", evalResult.Fluency,
		"completeness", evalResult.Completeness,
	)

	// ─── 3. 计算反馈级别 S/A/B/C ───
	score := evalResult.TotalScore
	feedbackLevel := calculateFeedbackLevel(ctx, s.repos, score)
	levelText := levelTextMap[feedbackLevel]
	logger.InfoContext(ctx, "evaluate mvp level", "level", feedbackLevel, "level_text", levelText)

	// ─── 4. 识别问题单词 ───
	var problemWords []string
	var worstWord string
	var worstWordScore float64 = 100
	wordDetails := make([]WordDetail, 0, len(evalResult.Words))

	for _, w := range evalResult.Words {
		isProblem := w.Score < 60
		wordDetails = append(wordDetails, WordDetail{
			Word:      w.Word,
			Score:     w.Score,
			IsProblem: isProblem,
		})
		if isProblem {
			problemWords = append(problemWords, w.Word)
		}
		if w.Score < worstWordScore {
			worstWordScore = w.Score
			worstWord = w.Word
		}
	}

	// ─── 5. LLM 生成反馈文本 ───
	systemPrompt, userMessage := buildPromptByLevel(feedbackLevel, targetText, score, worstWord, worstWordScore)
	feedbackText, err := s.llmProvider.Chat(ctx, systemPrompt, userMessage)
	if err != nil {
		logger.ErrorContext(ctx, "evaluate mvp llm failed", "error", err)
		feedbackText = levelText // fallback
	}
	logger.InfoContext(ctx, "evaluate mvp llm feedback", "feedback", feedbackText)

	// ─── 6. TTS 合成反馈音频 ───
	feedbackAudio, err := s.ttsProvider.Synthesize(ctx, feedbackText, nil)
	if err != nil {
		logger.ErrorContext(ctx, "evaluate mvp tts feedback failed", "error", err)
		return nil, fmt.Errorf("tts synthesize feedback failed: %w", err)
	}

	// ─── 7. 条件生成示范音频 ───
	var demoAudio *DemoAudio
	var demoAudioData []byte
	var demoText string
	var demoType string

	switch feedbackLevel {
	case "A", "B":
		// 问题单词示范
		if worstWord != "" {
			demoText = worstWord
			demoType = "word"
			demoAudioData, err = s.ttsProvider.Synthesize(ctx, worstWord, nil)
			if err != nil {
				logger.ErrorContext(ctx, "evaluate mvp tts demo word failed", "error", err)
			}
		}
	case "C":
		// 整句示范
		demoText = targetText
		demoType = "sentence"
		demoAudioData, err = s.ttsProvider.Synthesize(ctx, targetText, nil)
		if err != nil {
			logger.ErrorContext(ctx, "evaluate mvp tts demo sentence failed", "error", err)
		}
	}

	// ─── 8. 上传音频到 OSS ───
	evalID := uuid.New()
	var feedbackAudioURL string
	var demoAudioURL string

	if s.ossProvider != nil {
		// 上传反馈音频
		feedbackKey := fmt.Sprintf("evaluate/%s/feedback_%s.mp3", evalID, uuid.New())
		if url, uploadErr := s.ossProvider.UploadAudio(ctx, feedbackKey, feedbackAudio); uploadErr != nil {
			logger.ErrorContext(ctx, "evaluate mvp upload feedback audio failed", "error", uploadErr)
		} else {
			feedbackAudioURL = url
		}

		// 上传示范音频（如有）
		if len(demoAudioData) > 0 {
			demoKey := fmt.Sprintf("evaluate/%s/demo_%s.mp3", evalID, uuid.New())
			if url, uploadErr := s.ossProvider.UploadAudio(ctx, demoKey, demoAudioData); uploadErr != nil {
				logger.ErrorContext(ctx, "evaluate mvp upload demo audio failed", "error", uploadErr)
			} else {
				demoAudioURL = url
			}
		}
	}

	if len(demoAudioData) > 0 && demoAudioURL != "" {
		demoAudio = &DemoAudio{
			Type:     demoType,
			Text:     demoText,
			AudioURL: demoAudioURL,
		}
	}

	// ─── 9. 保存评测记录到数据库 ───
	if s.repos != nil {
		evaluation := &model.PronunciationEvaluation{
			ID:               evalID,
			UserID:           req.UserID,
			TargetText:       targetText,
			OverallScore:     int(score),
			AccuracyScore:    int(evalResult.Accuracy),
			FluencyScore:     int(evalResult.Fluency),
			IntegrityScore:   int(evalResult.Completeness),
			FeedbackLevel:    feedbackLevel,
			FeedbackText:     strPtr(feedbackText),
			FeedbackAudioURL: strPtr(feedbackAudioURL),
			ProblemWords:     model.StringArray(problemWords),
			DifficultyLevel:  req.DifficultyLevel,
			Status:           "completed",
		}
		if demoAudio != nil && demoType == "sentence" {
			evaluation.DemoSentenceAudioURL = strPtr(demoAudioURL)
		}
		if demoAudio != nil && demoType == "word" {
			evaluation.ProblemWordAudioURLs = model.StringMap{worstWord: demoAudioURL}
		}

		if saveErr := s.repos.PronunciationEvaluation.Create(ctx, evaluation); saveErr != nil {
			logger.ErrorContext(ctx, "evaluate mvp save db failed", "error", saveErr)
		} else {
			logger.InfoContext(ctx, "evaluate mvp saved to db", "eval_id", evalID)
		}
	}

	// ─── 10. 构建响应 ───
	resp := &EvaluateMVPResponse{
		OverallScore:     score,
		FeedbackLevel:    feedbackLevel,
		LevelText:        levelText,
		AccuracyScore:    evalResult.Accuracy,
		FluencyScore:     evalResult.Fluency,
		IntegrityScore:   evalResult.Completeness,
		FeedbackText:     feedbackText,
		FeedbackAudioURL: feedbackAudioURL,
		DemoAudio:        demoAudio,
		WordDetails:      wordDetails,
		TargetText:       targetText,
		EvalID:           evalID,
	}

	logger.InfoContext(ctx, "evaluate mvp completed", "eval_id", evalID, "level", feedbackLevel, "score", score)
	return resp, nil
}

// ===================== 辅助函数 =====================

// textIDMap 硬编码的文本 ID 映射（MVP 阶段）
var textIDMap = map[string]string{
	"text_000": "Hello, my name is Tom",
	"text_001": "The cat sat on the mat",
	"text_002": "I like to eat apples",
	"text_003": "She goes to school every day",
	"text_004": "The dog runs in the park",
	"text_005": "We are happy to see you",
	"text_006": "He reads books at night",
	"text_007": "They play games after school",
	"text_008": "My mother cooks dinner",
	"text_009": "The bird sings in the tree",
	"text_010": "I can swim very fast",
	"text_011": "She is a good student",
	"text_012": "They are playing in the park",
	"text_013": "He goes to the library",
	"text_014": "I like to eat pizza",
	"text_015": "We are learning English",
	"text_016": "The cat is sleeping",
	"text_017": "My father works in a hospital",
	"text_018": "They watch TV in the evening",
	"text_019": "I can play the piano",
	"text_020": "She writes stories",
}

// levelTextMap 反馈级别文本
var levelTextMap = map[string]string{
	"S": "Perfect!",
	"A": "Good Try!",
	"B": "Keep Going!",
	"C": "Let's Practice!",
}

// calculateFeedbackLevel 根据分数计算反馈级别
// 优先从数据库 system_settings 读取阈值，失败时使用默认值
func calculateFeedbackLevel(ctx context.Context, repos *db.Repositories, score float64) string {
	sMin := 90
	aMin := 70
	bMin := 50

	if repos != nil {
		if v, err := repos.SystemSetting.GetIntValue(ctx, "feedback_s_level_min_score"); err == nil {
			sMin = v
		}
		if v, err := repos.SystemSetting.GetIntValue(ctx, "feedback_a_level_min_score"); err == nil {
			aMin = v
		}
		if v, err := repos.SystemSetting.GetIntValue(ctx, "feedback_b_level_min_score"); err == nil {
			bMin = v
		}
	}

	switch {
	case score >= float64(sMin):
		return "S"
	case score >= float64(aMin):
		return "A"
	case score >= float64(bMin):
		return "B"
	default:
		return "C"
	}
}

// buildPromptByLevel 根据反馈级别构建 LLM Prompt
func buildPromptByLevel(level, targetText string, score float64, problemWord string, wordScore float64) (system string, user string) {
	switch level {
	case "S":
		return llmPrompts.BuildSLevelPrompt(targetText, score)
	case "A":
		return llmPrompts.BuildALevelPrompt(targetText, score, problemWord, wordScore)
	case "B":
		return llmPrompts.BuildBLevelPrompt(targetText, score, problemWord, wordScore)
	case "C":
		return llmPrompts.BuildCLevelPrompt(targetText, score)
	default:
		return llmPrompts.BuildCLevelPrompt(targetText, score)
	}
}

// strPtr 字符串指针辅助
func strPtr(s string) *string {
	return &s
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
