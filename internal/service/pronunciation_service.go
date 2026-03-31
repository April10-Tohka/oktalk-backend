package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/pkg/wav"
	"pronunciation-correction-system/internal/repository"
)

// PronunciationService 发音纠正 v2
type PronunciationService struct {
	loader       *config.PronunciationLoader
	sessionRepo  repository.PronunciationSessionRepository
	recordRepo   repository.PronunciationRecordRepository
	evalProvider domain.EvaluationProvider
	llmProvider  domain.LLMProvider
	ttsProvider  domain.TTSProvider
	ossProvider  domain.OSSProvider
	logger       *slog.Logger
}

// NewPronunciationService 构造函数（OSS 使用 domain.OSSProvider）
func NewPronunciationService(
	loader *config.PronunciationLoader,
	sessionRepo repository.PronunciationSessionRepository,
	recordRepo repository.PronunciationRecordRepository,
	eval domain.EvaluationProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	logger *slog.Logger,
) *PronunciationService {
	return &PronunciationService{
		loader:       loader,
		sessionRepo:  sessionRepo,
		recordRepo:   recordRepo,
		evalProvider: eval,
		llmProvider:  llm,
		ttsProvider:  tts,
		ossProvider:  oss,
		logger:       logger,
	}
}

// --- Unit list ---

// PronunciationUnitListItem 单元列表项
type PronunciationUnitListItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Topic      string `json:"topic"`
	Title      string `json:"title"`
	CoverEmoji string `json:"cover_emoji"`
	TotalItems int    `json:"total_items"`
}

// GetUnitList 获取单元列表（type 可选）
func (s *PronunciationService) GetUnitList(ctx context.Context, typeFilter string) []PronunciationUnitListItem {
	var units []*config.PronunciationUnitConfig
	if strings.TrimSpace(typeFilter) != "" {
		units = s.loader.GetByType(strings.TrimSpace(typeFilter))
	} else {
		units = s.loader.GetAll()
	}
	out := make([]PronunciationUnitListItem, 0, len(units))
	for _, u := range units {
		out = append(out, PronunciationUnitListItem{
			ID:         u.ID,
			Type:       u.Type,
			Topic:      u.Topic,
			Title:      u.Title,
			CoverEmoji: u.CoverEmoji,
			TotalItems: len(u.Items),
		})
	}
	return out
}

// --- Start session ---

// PronunciationStartSessionRequest 开始会话
type PronunciationStartSessionRequest struct {
	UserID string
	UnitID string
}

// StartSessionCurrentItem 当前练习项
type StartSessionCurrentItem struct {
	ID               int    `json:"id"`
	Content          string `json:"content"`
	StandardAudioURL string `json:"standard_audio_url"`
	ItemIndex        int    `json:"item_index"`
	TotalItems       int    `json:"total_items"`
	IsLast           bool   `json:"is_last"`
}

// PronunciationStartSessionResponse 开始会话响应
type PronunciationStartSessionResponse struct {
	SessionID   string                   `json:"session_id"`
	IsResumed   bool                     `json:"is_resumed"`
	CurrentItem *StartSessionCurrentItem `json:"current_item"`
}

// StartSession 开始或续学 Unit
func (s *PronunciationService) StartSession(ctx context.Context, req *PronunciationStartSessionRequest) (*PronunciationStartSessionResponse, error) {
	unit, ok := s.loader.GetByID(req.UnitID)
	if !ok {
		return nil, errHTTP(400, "Unit不存在")
	}
	total := len(unit.Items)
	if total == 0 {
		return nil, errHTTP(500, "单元配置异常")
	}

	var (
		sess       *model.PronunciationSession
		isResumed  bool
		currentIdx int
		sessionID  string
	)

	existing, err := s.sessionRepo.FindOngoingByUserAndUnit(ctx, req.UserID, req.UnitID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		isResumed = true
		sess = existing
		sessionID = existing.ID
		currentIdx = existing.CurrentIndex
	} else {
		sessionID = uuid.New()
		currentIdx = 1
		sess = &model.PronunciationSession{
			ID:           sessionID,
			UserID:       req.UserID,
			UnitID:       req.UnitID,
			CurrentIndex: 1,
			Status:       "ongoing",
		}
		if err := s.sessionRepo.Create(ctx, sess); err != nil {
			return nil, err
		}
	}

	item, ok := s.loader.GetItem(req.UnitID, currentIdx)
	if !ok {
		return nil, errHTTP(500, "单元条目配置异常")
	}

	return &PronunciationStartSessionResponse{
		SessionID: sessionID,
		IsResumed: isResumed,
		CurrentItem: &StartSessionCurrentItem{
			ID:               item.ID,
			Content:          item.Content,
			StandardAudioURL: item.StandardAudioURL,
			ItemIndex:        currentIdx,
			TotalItems:       total,
			IsLast:           currentIdx == total,
		},
	}, nil
}

// --- Evaluate ---

// PronunciationEvaluateRequest 评测请求
type PronunciationEvaluateRequest struct {
	UserID    string
	SessionID string
	ItemID    int
	AudioData []byte
	AudioType string
}

// EvaluationBlock 评测结果块
type EvaluationBlock struct {
	RawScore     float32  `json:"raw_score"`
	Stars        int      `json:"stars"`
	ProblemWords []string `json:"problem_words"`
}

// AIFeedbackBlock AI 反馈
type AIFeedbackBlock struct {
	Encourage  string `json:"encourage"`
	ProblemTip string `json:"problem_tip"`
	Suggestion string `json:"suggestion"`
	AiAudioURL   string `json:"ai_audio_url"`
}

// PronunciationEvaluateResponse 评测响应
type PronunciationEvaluateResponse struct {
	SessionID        string          `json:"session_id"`
	ItemID           int             `json:"item_id"`
	Content          string          `json:"content"`
	UserAudioURL     string          `json:"user_audio_url"`
	Evaluation       EvaluationBlock `json:"evaluation"`
	AIFeedback       AIFeedbackBlock `json:"ai_feedback"`
	RecommendAdvance bool            `json:"recommend_advance"`
}

// Evaluate 提交录音并评测
func (s *PronunciationService) Evaluate(ctx context.Context, req *PronunciationEvaluateRequest) (*PronunciationEvaluateResponse, error) {
	if req.SessionID == "" || req.ItemID <= 0 {
		return nil, errHTTP(400, "参数无效")
	}
	if len(req.AudioData) == 0 {
		return nil, errHTTP(400, "音频为空")
	}
	const maxAudio = 10 * 1024 * 1024
	if len(req.AudioData) > maxAudio {
		return nil, errHTTP(400, "音频文件过大")
	}

	sess, err := s.sessionRepo.FindByID(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errHTTP(404, "会话不存在")
	}
	if sess.UserID != req.UserID {
		return nil, errHTTP(403, "forbidden")
	}
	if sess.Status != "ongoing" {
		return nil, errHTTP(400, "Unit已完成")
	}
	if sess.CurrentIndex != req.ItemID {
		return nil, errHTTP(400, "步骤不匹配")
	}

	unit, ok := s.loader.GetByID(sess.UnitID)
	if !ok {
		return nil, errHTTP(500, "单元配置异常")
	}
	item, ok := s.loader.GetItem(sess.UnitID, req.ItemID)
	if !ok {
		return nil, errHTTP(500, "数据异常")
	}

	category := xunfeiCategory(unit.Type)
	practiceType := unit.Type
	if practiceType == "" {
		practiceType = "word"
	}
	// 科大讯飞语音评测接口要求wav格式的音频要转换为pcm格式
	audioData, err := wav.StripToLinearPCM(req.AudioData)
	if err != nil {
		return nil, errHTTP(400, "音频格式无效：请使用 WAV，或 16kHz/16bit/mono 裸 PCM")
	}
	evalResult, err := s.evalProvider.Assess(ctx, item.Content, audioData, category)
	if err != nil {
		s.logger.Error("pronunciation v2 assess failed", slog.String("error", err.Error()))
		return nil, errHTTP(500, "评测服务异常")
	}

	rawScore := float32(evalResult.TotalScore)
	rawScore = clampFloat32(rawScore, 0, 5)
	stars := int(math.Round(float64(rawScore)))
	stars = clampInt(stars, 0, 5)

	problemWords := collectProblemWords(evalResult.Words)

	userAudioURL := ""
	key := fmt.Sprintf("pronunciation/%s/%d_%s.wav", req.SessionID, req.ItemID, uuid.New())
	if url, upErr := s.ossProvider.UploadAudio(ctx, key, req.AudioData); upErr != nil {
		s.logger.Warn("pronunciation v2 user audio upload failed", slog.String("error", upErr.Error()))
	} else {
		userAudioURL = url
	}

	llmOut := s.buildFeedbackLLM(ctx, item.Content, practiceType, rawScore, stars, problemWords)

	ttsText := strings.TrimSpace(llmOut.Encourage + " " + llmOut.ProblemTip + " " + llmOut.Suggestion)
	aiAudioURL := ""
	if ttsText != "" {
		opts := domain.DefaultSynthesizeOptions()
		opts.Format = "wav"
		audio, ttsErr := s.ttsProvider.Synthesize(ctx, ttsText, opts)
		if ttsErr != nil {
			s.logger.Warn("pronunciation v2 TTS failed", slog.String("error", ttsErr.Error()))
		} else {
			aiKey := fmt.Sprintf("pronunciation/%s/ai_%s.wav", req.SessionID, uuid.New())
			if u, e2 := s.ossProvider.UploadAudio(ctx, aiKey, audio); e2 != nil {
				s.logger.Warn("pronunciation v2 AI audio upload failed", slog.String("error", e2.Error()))
			} else {
				aiAudioURL = u
			}
		}
	}

	pwJSON, _ := json.Marshal(problemWords)
	rec := &model.PronunciationRecord{
		ID:            uuid.New(),
		SessionID:     req.SessionID,
		UserID:        req.UserID,
		UnitID:        sess.UnitID,
		ItemID:        req.ItemID,
		Content:       item.Content,
		PracticeType:  practiceType,
		RawScore:      rawScore,
		Stars:         stars,
		ProblemWords:  string(pwJSON),
		UserAudioURL:  userAudioURL,
		AIEncourage:   llmOut.Encourage,
		AIProblemTip:  llmOut.ProblemTip,
		AISuggestion:  llmOut.Suggestion,
		AIAudioURL:    aiAudioURL,
		IsRejected:    evalResult.IsRejected,
		AccuracyScore: clampFloat32(float32(evalResult.AccuracyScore), 0, 5),
		Fluency:       clampFloat32(float32(evalResult.FluencyScore), 0, 5),
		Integrity:     clampFloat32(float32(evalResult.IntegrityScore), 0, 5),
		StandardScore: clampFloat32(float32(evalResult.StandardScore), 0, 5),
	}
	go func(r *model.PronunciationRecord) {
		ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.recordRepo.Create(ctx2, r); err != nil {
			s.logger.Warn("pronunciation v2 record persist failed", slog.String("error", err.Error()))
		}
	}(rec)

	recommend := rawScore >= 3.75

	return &PronunciationEvaluateResponse{
		SessionID: req.SessionID,
		ItemID:    req.ItemID,
		Content:   item.Content,
		UserAudioURL: userAudioURL,
		Evaluation: EvaluationBlock{
			RawScore:     rawScore,
			Stars:        stars,
			ProblemWords: problemWords,
		},
		AIFeedback: AIFeedbackBlock{
			Encourage:  llmOut.Encourage,
			ProblemTip: llmOut.ProblemTip,
			Suggestion: llmOut.Suggestion,
			AiAudioURL:   aiAudioURL,
		},
		RecommendAdvance: recommend,
	}, nil
}

func xunfeiCategory(unitType string) string {
	switch strings.ToLower(strings.TrimSpace(unitType)) {
	case "sentence":
		return "read_sentence"
	default:
		return "read_word"
	}
}

// collectProblemWords 依据 domain.WordEvaluationResult.DpMessage：非 0 表示增漏读等问题
func collectProblemWords(words []domain.WordEvaluationResult) []string {
	var out []string
	for _, w := range words {
		if w.DpMessage != 0 && strings.TrimSpace(w.Word) != "" {
			out = append(out, w.Word)
		}
	}
	return out
}

type llmFeedbackJSON struct {
	Encourage  string `json:"encourage"`
	ProblemTip string `json:"problem_tip"`
	Suggestion string `json:"suggestion"`
}

func (s *PronunciationService) buildFeedbackLLM(ctx context.Context, content, practiceType string, rawScore float32, stars int, problemWords []string) llmFeedbackJSON {
	sys := `你是一位儿童英语发音老师，说话简短活泼，多用 emoji，语气鼓励正向。
只能返回 JSON，不允许返回任何其他内容。`
	usr := fmt.Sprintf(`小朋友正在练习英语发音。
练习内容：%s（%s）
总体评分：%.1f/5，对应%d颗星
有发音问题的词：%v（空列表表示全部正确）

请生成一段发音反馈，只返回如下 JSON：
{"encourage":"鼓励语，一句话，10词以内","problem_tip":"问题提示，一句话；无问题词时写 Everything sounds great!","suggestion":"建议，一句话，简单可操作"}`,
		content, practiceType, rawScore, stars, problemWords)

	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	if err != nil {
		s.logger.Warn("pronunciation v2 feedback LLM failed", slog.String("error", err.Error()))
		return fallbackFeedback(stars)
	}
	js := extractJSONObject(raw)
	var out llmFeedbackJSON
	if err := json.Unmarshal([]byte(js), &out); err != nil || out.Encourage == "" {
		s.logger.Warn("pronunciation v2 feedback JSON parse failed", slog.String("raw", raw))
		return fallbackFeedback(stars)
	}
	return out
}

func fallbackFeedback(stars int) llmFeedbackJSON {
	if stars >= 4 {
		return llmFeedbackJSON{
			Encourage:  "Great job! 🌟",
			ProblemTip: "Everything sounds great!",
			Suggestion: "Keep it up!",
		}
	}
	return llmFeedbackJSON{
		Encourage:  "Good try! 💪",
		ProblemTip: "Let's practice more!",
		Suggestion: "Listen and try again! 🎧",
	}
}

func clampFloat32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// --- Advance ---

// PronunciationAdvanceRequest 推进请求
type PronunciationAdvanceRequest struct {
	UserID        string
	SessionID     string
	CurrentItemID int
}

// PronunciationAdvanceResponse 推进响应
type PronunciationAdvanceResponse struct {
	UnitCompleted bool                     `json:"unit_completed"`
	NextItem      *StartSessionCurrentItem `json:"next_item,omitempty"`
}

// Advance 进入下一项或结束单元
func (s *PronunciationService) Advance(ctx context.Context, req *PronunciationAdvanceRequest) (*PronunciationAdvanceResponse, error) {
	sess, err := s.sessionRepo.FindByID(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errHTTP(404, "会话不存在")
	}
	if sess.UserID != req.UserID {
		return nil, errHTTP(403, "forbidden")
	}
	if sess.Status != "ongoing" {
		return nil, errHTTP(400, "Unit已完成")
	}
	if sess.CurrentIndex != req.CurrentItemID {
		return nil, errHTTP(400, "步骤不匹配")
	}

	unit, ok := s.loader.GetByID(sess.UnitID)
	if !ok {
		return nil, errHTTP(500, "单元配置异常")
	}
	totalItems := len(unit.Items)
	if totalItems == 0 {
		return nil, errHTTP(500, "单元配置异常")
	}

	isLast := req.CurrentItemID == totalItems
	if isLast {
		if err := s.sessionRepo.UpdateStatus(ctx, req.SessionID, "finished"); err != nil {
			return nil, err
		}
		return &PronunciationAdvanceResponse{UnitCompleted: true, NextItem: nil}, nil
	}

	nextIndex := req.CurrentItemID + 1
	if err := s.sessionRepo.UpdateCurrentIndex(ctx, req.SessionID, nextIndex); err != nil {
		return nil, err
	}
	nextItem, ok := s.loader.GetItem(sess.UnitID, nextIndex)
	if !ok {
		return nil, errHTTP(500, "单元条目配置异常")
	}

	return &PronunciationAdvanceResponse{
		UnitCompleted: false,
		NextItem: &StartSessionCurrentItem{
			ID:               nextItem.ID,
			Content:          nextItem.Content,
			StandardAudioURL: nextItem.StandardAudioURL,
			ItemIndex:        nextIndex,
			TotalItems:       totalItems,
			IsLast:           nextIndex == totalItems,
		},
	}, nil
}

// --- Summary ---

// summaryItemRow 总结中单条
type summaryItemRow struct {
	Content   string  `json:"content"`
	BestScore float32 `json:"best_score"`
	BestStars int     `json:"best_stars"`
}

// PronunciationSummaryResponse 单元总结
type PronunciationSummaryResponse struct {
	UnitTitle    string           `json:"unit_title"`
	AverageScore float32          `json:"average_score"`
	AverageStars int              `json:"average_stars"`
	Items        []summaryItemRow `json:"items"`
	SummaryLLM   string           `json:"summary"`
	Highlight    []string         `json:"highlight"`
	Weak         []string         `json:"weak"`
}

// GetSummary 获取已完成单元的总结
func (s *PronunciationService) GetSummary(ctx context.Context, userID, sessionID string) (*PronunciationSummaryResponse, error) {
	sess, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errHTTP(404, "会话不存在")
	}
	if sess.UserID != userID {
		return nil, errHTTP(403, "forbidden")
	}
	if sess.Status != "finished" {
		return nil, errHTTP(400, "Unit未完成，无法查看总结")
	}

	unit, ok := s.loader.GetByID(sess.UnitID)
	if !ok {
		return nil, errHTTP(500, "单元配置异常")
	}

	bestMap, err := s.recordRepo.GetBestScorePerItem(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	items := make([]summaryItemRow, 0, len(unit.Items))
	var sum float64
	for _, it := range unit.Items {
		sc := bestMap[it.ID]
		st := int(math.Round(float64(sc)))
		st = clampInt(st, 0, 5)
		items = append(items, summaryItemRow{
			Content:   it.Content,
			BestScore: sc,
			BestStars: st,
		})
		sum += float64(sc)
	}
	n := len(unit.Items)
	avg := float32(0)
	if n > 0 {
		avg = float32(sum / float64(n))
	}
	avgStars := int(math.Round(float64(avg)))
	avgStars = clampInt(avgStars, 0, 5)

	// LLM 用 items 列表
	itemsForLLM := make([]map[string]interface{}, 0, len(items))
	for _, row := range items {
		itemsForLLM = append(itemsForLLM, map[string]interface{}{
			"content":    row.Content,
			"best_score": row.BestScore,
		})
	}
	itemsJSON, _ := json.Marshal(itemsForLLM)

	sys := `你是一位儿童英语发音老师，说话简短活泼，多用 emoji，语气鼓励正向。
只能返回 JSON，不允许返回任何其他内容。`
	usr := fmt.Sprintf(`小朋友完成了一组英语发音练习，以下是每个内容的最佳得分：
%s
平均分：%.1f/5

请生成练习总结，只返回如下 JSON：
{"summary":"总结语，两句话以内，鼓励为主","highlight":["发音较好的内容列表"],"weak":["需加强的内容列表，无则为空数组"]}`,
		string(itemsJSON), avg)

	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	var summaryText string
	var highlight, weak []string
	if err != nil {
		s.logger.Warn("pronunciation v2 summary LLM failed", slog.String("error", err.Error()))
		summaryText = "Great effort! 🎉 Keep practicing every day!"
		highlight = []string{}
		weak = []string{}
	} else {
		js := extractJSONObject(raw)
		var wrap struct {
			Summary   string   `json:"summary"`
			Highlight []string `json:"highlight"`
			Weak      []string `json:"weak"`
		}
		if err := json.Unmarshal([]byte(js), &wrap); err != nil || wrap.Summary == "" {
			summaryText = "Great effort! 🎉 Keep practicing every day!"
			highlight = []string{}
			weak = []string{}
		} else {
			summaryText = wrap.Summary
			highlight = wrap.Highlight
			weak = wrap.Weak
		}
	}
	if highlight == nil {
		highlight = []string{}
	}
	if weak == nil {
		weak = []string{}
	}

	return &PronunciationSummaryResponse{
		UnitTitle:    unit.Title,
		AverageScore: avg,
		AverageStars: avgStars,
		Items:        items,
		SummaryLLM:   summaryText,
		Highlight:    highlight,
		Weak:         weak,
	}, nil
}
