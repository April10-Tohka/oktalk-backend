// Package service 提供 AI 发音纠正业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	llmPrompts "pronunciation-correction-system/internal/infrastructure/llm"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/worker"
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
	AudioData       []byte
	AudioType       string
	TextID          string
	Category        string // read_word / read_sentence
	DifficultyLevel string // beginner / intermediate / advanced
	UserID          string
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
	Message          string            `json:"message,omitempty"`
	TextID           string            `json:"text_id,omitempty"`
	ReferenceText    string            `json:"reference_text,omitempty"`
	OverallScore     float64           `json:"overall_score,omitempty"`
	Scores           *EvalScores       `json:"scores,omitempty"`
	DurationMs       int               `json:"duration_ms,omitempty"`
	ProblemWords     []string          `json:"problem_words,omitempty"`
	DetailedFeedback *DetailedFeedback `json:"detailed_feedback,omitempty"`
	ReferenceAudio   string            `json:"reference_audio,omitempty"`
	CreatedAt        string            `json:"created_at,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
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

// ===== Service 实现 =====

// evaluateServiceImpl Evaluate Service 实现
type evaluateServiceImpl struct {
	repos              *db.Repositories
	evaluationProvider domain.EvaluationProvider
	llmProvider        domain.LLMProvider
	ttsProvider        domain.TTSProvider
	ossProvider        domain.OSSProvider
	taskCache          *cache.TaskCache
	evalCache          *cache.EvalCache
	workerManager      *worker.Manager
	logger             *slog.Logger
}

// NewEvaluateService 创建 EvaluateService
func NewEvaluateService(
	repos *db.Repositories,
	evaluationProvider domain.EvaluationProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	ossProvider domain.OSSProvider,
	taskCache *cache.TaskCache,
	evalCache *cache.EvalCache,
	workerMgr *worker.Manager,
	logger *slog.Logger,
) EvaluateService {
	return &evaluateServiceImpl{
		repos:              repos,
		evaluationProvider: evaluationProvider,
		llmProvider:        llmProvider,
		ttsProvider:        ttsProvider,
		ossProvider:        ossProvider,
		taskCache:          taskCache,
		evalCache:          evalCache,
		workerManager:      workerMgr,
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
	evalResult, err := s.evaluationProvider.Assess(ctx, targetText, req.AudioData, req.Category)
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
	// 步骤 1：基础校验
	if req == nil {
		return "", errors.New("submit evaluation request is nil")
	}
	if len(req.AudioData) == 0 {
		return "", errors.New("audio data is empty")
	}
	if req.UserID == "" {
		return "", errors.New("user id is empty")
	}
	if req.TextID == "" {
		return "", errors.New("text_id is required")
	}

	// 步骤 2：校验 category
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "read_sentence"
	}
	if category != "read_word" && category != "read_sentence" {
		return "", fmt.Errorf("invalid category: %s (must be read_word or read_sentence)", category)
	}

	// 步骤 3：处理默认参数
	difficultyLevel := strings.TrimSpace(req.DifficultyLevel)
	if difficultyLevel == "" {
		difficultyLevel = "beginner"
	}
	audioType := strings.ToLower(strings.TrimSpace(req.AudioType))
	if audioType == "" {
		audioType = "wav"
	}

	// 步骤 4：查询朗读文本
	targetText, ok := textIDMap[req.TextID]
	if !ok {
		// TODO: 从数据库查询 text_id
		return "", fmt.Errorf("text not found: %s", req.TextID)
	}

	// 步骤 5：WAV → PCM（去掉 44 字节 header）
	audioData := req.AudioData
	if audioType == "wav" && len(audioData) > 44 {
		audioData = audioData[44:]
	}

	// 步骤 6：序列化 Payload
	type evalPayload struct {
		AudioData       []byte `json:"audio_data"`
		AudioType       string `json:"audio_type"`
		TextID          string `json:"text_id"`
		TargetText      string `json:"target_text"`
		Category        string `json:"category"`
		DifficultyLevel string `json:"difficulty_level"`
		UserID          string `json:"user_id"`
	}
	payload, err := json.Marshal(&evalPayload{
		AudioData:       audioData,
		AudioType:       audioType,
		TextID:          req.TextID,
		TargetText:      targetText,
		Category:        category,
		DifficultyLevel: difficultyLevel,
		UserID:          req.UserID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal eval payload: %w", err)
	}

	// 步骤 7：构建 Task 并提交
	task := &worker.Task{
		Type:    "evaluate",
		UserID:  req.UserID,
		Payload: payload,
	}

	taskID, err := s.workerManager.SubmitTask(ctx, task)
	if err != nil {
		logger.ErrorContext(ctx, "submit eval task failed", "error", err)
		return "", fmt.Errorf("submit task: %w", err)
	}

	// 步骤 8：更新学习进度（占位）
	logger.InfoContext(ctx, "eval task submitted",
		"task_id", taskID, "user_id", req.UserID, "text_id", req.TextID)

	return taskID, nil
}

// evalStageDescriptions 评测阶段描述
var evalStageDescriptions = map[string]string{
	"queued":     "评测任务已进入队列，等待处理",
	"evaluating": "正在分析音素...",
	"analyzing":  "正在生成反馈...",
	"tts":        "正在合成音频...",
	"oss":        "正在上传音频...",
	"db":         "正在保存记录...",
	"completed":  "评测完成",
}

func (s *evaluateServiceImpl) GetEvaluationResult(ctx context.Context, evalID string) (*EvaluationResultResponse, error) {
	if evalID == "" {
		return nil, errors.New("eval_id is empty")
	}

	// 步骤 1：从缓存查询任务状态
	meta, err := s.taskCache.GetTaskMeta(ctx, evalID)
	if err != nil {
		logger.ErrorContext(ctx, "get eval task meta failed", "eval_id", evalID, "error", err)
		return nil, fmt.Errorf("query task status: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("evaluation not found: %s", evalID)
	}

	// 步骤 2：根据状态构建响应
	switch meta.Status {
	case "pending":
		stage := meta.CurrentStage
		if stage == "" {
			stage = "queued"
		}
		return &EvaluationResultResponse{
			EvalID:  evalID,
			Status:  "pending",
			Message: evalStageDescriptions[stage],
		}, nil

	case "processing":
		stage := meta.CurrentStage
		msg := evalStageDescriptions[stage]
		if msg == "" {
			msg = "正在处理中..."
		}
		return &EvaluationResultResponse{
			EvalID:  evalID,
			Status:  "processing",
			Message: msg,
		}, nil

	case "success":
		// 从 evalCache 获取完整结果
		resultKey := meta.ResultKey
		var resultEvalID string
		fmt.Sscanf(resultKey, "evaluate:result:%s", &resultEvalID)
		if resultEvalID == "" {
			resultEvalID = evalID
		}

		evalResult, found, cacheErr := s.evalCache.GetEvalResult(ctx, resultEvalID)
		if cacheErr != nil {
			logger.ErrorContext(ctx, "get eval result from cache failed", "eval_id", evalID, "error", cacheErr)
			return nil, fmt.Errorf("get eval result: %w", cacheErr)
		}
		if !found {
			return nil, fmt.Errorf("eval result not found in cache for: %s", evalID)
		}

		// 构建单词详情
		var problemWords []string
		for _, wd := range evalResult.WordDetails {
			if wd.IsProblem {
				problemWords = append(problemWords, wd.Word)
			}
		}

		resp := &EvaluationResultResponse{
			EvalID:        evalID,
			Status:        "success",
			TextID:        evalResult.TextID,
			ReferenceText: evalResult.TargetText,
			OverallScore:  evalResult.OverallScore,
			Scores: &EvalScores{
				Pronunciation: evalResult.AccuracyScore,
				Fluency:       evalResult.FluencyScore,
				Integrity:     evalResult.IntegrityScore,
			},
			ProblemWords:   problemWords,
			ReferenceAudio: evalResult.AudioURL,
			CreatedAt:      time.Unix(evalResult.CreatedAt, 0).Format(time.RFC3339),
		}

		return resp, nil

	case "failed":
		return &EvaluationResultResponse{
			EvalID:       evalID,
			Status:       "failed",
			ErrorMessage: meta.Error,
			Message:      "评测处理失败，请重试或联系支持",
		}, nil

	default:
		logger.ErrorContext(ctx, "unknown eval task status", "eval_id", evalID, "status", meta.Status)
		return nil, fmt.Errorf("unknown task status: %s", meta.Status)
	}
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

// ===================== EvalTaskProcessor =====================

// evalPayloadData 异步评测任务 payload 反序列化目标
type evalPayloadData struct {
	AudioData       []byte `json:"audio_data"`
	AudioType       string `json:"audio_type"`
	TextID          string `json:"text_id"`
	TargetText      string `json:"target_text"`
	Category        string `json:"category"`
	DifficultyLevel string `json:"difficulty_level"`
	UserID          string `json:"user_id"`
}

// EvalTaskProcessor 实现 worker.TaskProcessor 接口
type EvalTaskProcessor struct {
	evaluationProvider domain.EvaluationProvider
	llmProvider        domain.LLMProvider
	ttsProvider        domain.TTSProvider
	ossProvider        domain.OSSProvider
	repos              *db.Repositories
	taskCache          *cache.TaskCache
	logger             *slog.Logger
}

// NewEvalTaskProcessor 创建 EvalTaskProcessor
func NewEvalTaskProcessor(
	eval domain.EvaluationProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	repos *db.Repositories,
	taskCache *cache.TaskCache,
	logger *slog.Logger,
) *EvalTaskProcessor {
	return &EvalTaskProcessor{
		evaluationProvider: eval,
		llmProvider:        llm,
		ttsProvider:        tts,
		ossProvider:        oss,
		repos:              repos,
		taskCache:          taskCache,
		logger:             logger,
	}
}

// Process 实现 worker.TaskProcessor 接口
// 返回: (*cache.EvalResult, resultKey, error)
func (p *EvalTaskProcessor) Process(ctx context.Context, task *worker.Task) (interface{}, string, error) {
	// 反序列化 payload
	var payload evalPayloadData
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return nil, "", fmt.Errorf("unmarshal eval payload: %w", err)
	}

	resultKey := fmt.Sprintf(cache.KeyEvalResult, task.ID)
	targetText := payload.TargetText
	category := payload.Category

	// ===== A1: 讯飞语音评测 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "evaluating")

	var evalResult *domain.EvaluationResult
	var evalErr error
	for retry := 0; retry < 3; retry++ {
		evalResult, evalErr = p.evaluationProvider.Assess(ctx, targetText, payload.AudioData, category)
		if evalErr == nil {
			break
		}
		p.logger.Warn("Evaluation retry", slog.Int("attempt", retry+1), slog.String("error", evalErr.Error()))
		time.Sleep(time.Duration(retry+1) * time.Second)
	}
	if evalErr != nil {
		return nil, resultKey, fmt.Errorf("[evaluating] %w", evalErr)
	}

	// ===== A2: 后处理和分析评测结果 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "analyzing")

	score := evalResult.TotalScore
	feedbackLevel := calculateFeedbackLevel(ctx, p.repos, score)

	var problemWords []string
	var worstWord string
	var worstWordScore float64 = 100
	wordDetails := make([]cache.EvalWordDetail, 0, len(evalResult.Words))

	for _, w := range evalResult.Words {
		isProblem := w.Score < 60
		wordDetails = append(wordDetails, cache.EvalWordDetail{
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

	// ===== A3: LLM 生成分级反馈文本 =====
	systemPrompt, userMessage := buildPromptByLevel(feedbackLevel, targetText, score, worstWord, worstWordScore)
	feedbackText, llmErr := p.llmProvider.Chat(ctx, systemPrompt, userMessage)
	if llmErr != nil {
		p.logger.Warn("Eval LLM failed, using fallback", slog.String("error", llmErr.Error()))
		feedbackText = levelTextMap[feedbackLevel]
	}
	if feedbackText == "" {
		feedbackText = levelTextMap[feedbackLevel]
	}

	// ===== A4: TTS 合成反馈音频 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "tts")

	var feedbackAudio []byte
	var ttsErr error
	for retry := 0; retry < 2; retry++ {
		feedbackAudio, ttsErr = p.ttsProvider.Synthesize(ctx, feedbackText, nil)
		if ttsErr == nil {
			break
		}
		p.logger.Warn("Eval TTS feedback retry", slog.Int("attempt", retry+1), slog.String("error", ttsErr.Error()))
		time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
	}
	if ttsErr != nil {
		p.logger.Error("Eval TTS feedback failed", slog.String("error", ttsErr.Error()))
	}

	// ===== A5: 条件生成示范音频 =====
	var demoAudioData []byte
	var demoType, demoText string

	switch feedbackLevel {
	case "A", "B":
		if worstWord != "" {
			demoType = "word"
			demoText = worstWord
			demoAudioData, _ = p.ttsProvider.Synthesize(ctx, worstWord, nil)
		}
	case "C":
		demoType = "sentence"
		demoText = targetText
		demoAudioData, _ = p.ttsProvider.Synthesize(ctx, targetText, nil)
	}

	// ===== A6: 上传音频到 OSS =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "oss")

	evalID := task.ID
	var feedbackAudioURL, demoAudioURL string

	if p.ossProvider != nil {
		if len(feedbackAudio) > 0 {
			fbKey := fmt.Sprintf("evaluate/%s/feedback_%s.mp3", evalID, uuid.New())
			if url, uploadErr := p.ossProvider.UploadAudio(ctx, fbKey, feedbackAudio); uploadErr != nil {
				p.logger.Error("upload eval feedback audio failed", slog.String("error", uploadErr.Error()))
			} else {
				feedbackAudioURL = url
			}
		}
		if len(demoAudioData) > 0 {
			demoKey := fmt.Sprintf("evaluate/%s/demo_%s.mp3", evalID, uuid.New())
			if url, uploadErr := p.ossProvider.UploadAudio(ctx, demoKey, demoAudioData); uploadErr != nil {
				p.logger.Error("upload eval demo audio failed", slog.String("error", uploadErr.Error()))
			} else {
				demoAudioURL = url
			}
		}
	}

	// ===== A7: 保存评测结果到数据库 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "db")

	if p.repos != nil {
		evaluation := &model.PronunciationEvaluation{
			ID:               evalID,
			UserID:           task.UserID,
			TargetText:       targetText,
			OverallScore:     int(score),
			AccuracyScore:    int(evalResult.Accuracy),
			FluencyScore:     int(evalResult.Fluency),
			IntegrityScore:   int(evalResult.Completeness),
			FeedbackLevel:    feedbackLevel,
			FeedbackText:     strPtr(feedbackText),
			FeedbackAudioURL: strPtr(feedbackAudioURL),
			ProblemWords:     model.StringArray(problemWords),
			DifficultyLevel:  payload.DifficultyLevel,
			Status:           "completed",
		}
		if demoType == "sentence" && demoAudioURL != "" {
			evaluation.DemoSentenceAudioURL = strPtr(demoAudioURL)
		}
		if demoType == "word" && worstWord != "" && demoAudioURL != "" {
			evaluation.ProblemWordAudioURLs = model.StringMap{worstWord: demoAudioURL}
		}

		for dbRetry := 0; dbRetry < 3; dbRetry++ {
			if saveErr := p.repos.PronunciationEvaluation.Create(ctx, evaluation); saveErr != nil {
				p.logger.Error("save eval db retry", slog.Int("attempt", dbRetry+1), slog.String("error", saveErr.Error()))
				time.Sleep(time.Duration(dbRetry+1) * 500 * time.Millisecond)
				continue
			}
			break
		}
	}

	// ===== A8: 更新学习统计（占位） =====
	p.logger.Info("eval completion progress updated (placeholder)",
		slog.String("task_id", task.ID),
		slog.String("user_id", task.UserID),
		slog.Float64("score", score),
	)

	// ===== A9: 构建结果 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "completed")

	result := &cache.EvalResult{
		EvalID:         evalID,
		UserID:         task.UserID,
		TextID:         payload.TextID,
		TargetText:     targetText,
		OverallScore:   score,
		AccuracyScore:  evalResult.Accuracy,
		FluencyScore:   evalResult.Fluency,
		IntegrityScore: evalResult.Completeness,
		FeedbackLevel:  feedbackLevel,
		FeedbackText:   feedbackText,
		AudioURL:       feedbackAudioURL,
		DemoAudioURL:   demoAudioURL,
		DemoType:       demoType,
		DemoText:       demoText,
		ProblemWords:   problemWords,
		WordDetails:    wordDetails,
		CreatedAt:      time.Now().Unix(),
	}

	return result, resultKey, nil
}

// ===================== EvalResultPersister =====================

// EvalResultPersister 实现 worker.ResultPersister 接口
type EvalResultPersister struct {
	logger *slog.Logger
}

// NewEvalResultPersister 创建 EvalResultPersister
func NewEvalResultPersister(logger *slog.Logger) *EvalResultPersister {
	return &EvalResultPersister{logger: logger}
}

// SaveResult 将结果持久化到数据库
// EvalTaskProcessor.Process 中已经完成了 DB 写入，这里只做日志记录
func (p *EvalResultPersister) SaveResult(ctx context.Context, task *worker.Task, result interface{}) error {
	p.logger.Info("eval result persisted (already saved in processor)",
		slog.String("task_id", task.ID),
		slog.String("type", task.Type),
	)
	return nil
}
