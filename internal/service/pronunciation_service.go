package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
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
	cache        repository.SceneCacheRepository // Redis 缓存，复用场景缓存接口
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
	cache repository.SceneCacheRepository,
) *PronunciationService {
	return &PronunciationService{
		loader:       loader,
		sessionRepo:  sessionRepo,
		recordRepo:   recordRepo,
		evalProvider: eval,
		llmProvider:  llm,
		ttsProvider:  tts,
		ossProvider:  oss,
		cache:        cache,
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
	RawScore     float64  `json:"raw_score"`
	Stars        int      `json:"stars"`
	ProblemWords []string `json:"problem_words"`
}

// AIFeedbackBlock AI 反馈
type AIFeedbackBlock struct {
	Encourage  string `json:"encourage"`
	ProblemTip string `json:"problem_tip"`
	HowToFix   string `json:"how_to_fix"`
	AiAudioURL string `json:"ai_audio_url"`
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

	// 判断评测是否通过
	if evalResult.IsRejected {
		// 判断except_info属性值
		// except_info=28673时，16进制为0x7001，表示引擎判断该语音为无语音或音量小类型
		// except_info=28676时，16进制为0x7004，表示引擎判断该语音为乱说类型
		// except_info=28680时，16进制为0x7008，表示引擎判断该语音为信噪比低类型
		// except_info=28690时，16进制为0x7012，表示引擎判断该语音为截幅类型
		// except_info=28689时，16进制为0x7011，表示引擎判断没有音频输入，请检测音频或录音设备是否正常
		switch evalResult.ExceptInfo {
		case 28673:
			// 原逻辑：无语音或音量小类型
			return nil, errHTTP(400, "好像没听到你的声音哦，嘴巴靠近一点，大声读出来吧！")
		case 28676:
			// 原逻辑：乱说类型
			return nil, errHTTP(400, "这次好像读得不太对哦，跟着示范再认真试一次吧！")
		case 28680:
			// 原逻辑：信噪比低类型（通常是因为周围太吵）
			return nil, errHTTP(400, "周围好像有点吵呀，找一个安静的地方，我们再试一次！")
		case 28690:
			// 原逻辑：截幅类型（声音太大爆音了）
			return nil, errHTTP(400, "你的声音太洪亮啦，把手机拿远一点点，轻轻读出来就好啦！")
		case 28689:
			// 原逻辑：没有音频输入，请检测音频或录音设备是否正常
			return nil, errHTTP(400, "检查一下手机有没有开麦克风权限哦！")
		}

	}

	rawScore := evalResult.TotalScore
	rawScore = clampFloat64(rawScore, 0, 5)
	stars := int(math.Round(rawScore))
	stars = clampInt(stars, 0, 5)

	problemWords := collectProblemWords(evalResult.Words)
	correctionInput := buildCorrectionInput(item.Content, practiceType, evalResult)

	// 上传用户音频到oss
	userAudioURL := ""
	go func() {
		key := fmt.Sprintf("pronunciation/%s/%d_user_%s.wav", req.SessionID, req.ItemID, uuid.New())
		if url, upErr := s.ossProvider.UploadAudio(ctx, key, req.AudioData); upErr != nil {
			s.logger.Warn("pronunciation v2 user audio upload failed", slog.String("error", upErr.Error()))
		} else {
			userAudioURL = url
		}
	}()

	llmOut := s.buildFeedbackLLMWithCache(ctx, sess.UnitID, req.ItemID, item.Content, practiceType, rawScore, stars, correctionInput)

	// tts
	ttsText := strings.TrimSpace(llmOut.Encourage + " " + llmOut.ProblemTip + " " + llmOut.Retry)
	aiAudioURL := s.synthPronAudioWithCache(ctx, req.SessionID, ttsText)

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
		AIHowToFix:    llmOut.HowToFix,
		AIAudioURL:    aiAudioURL,
		IsRejected:    evalResult.IsRejected,
		AccuracyScore: clampFloat64(evalResult.AccuracyScore, 0, 5),
		Fluency:       clampFloat64(evalResult.FluencyScore, 0, 5),
		Integrity:     clampFloat64(evalResult.IntegrityScore, 0, 5),
		StandardScore: clampFloat64(evalResult.StandardScore, 0, 5),
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
		SessionID:    req.SessionID,
		ItemID:       req.ItemID,
		Content:      item.Content,
		UserAudioURL: userAudioURL,
		Evaluation: EvaluationBlock{
			RawScore:     rawScore,
			Stars:        stars,
			ProblemWords: problemWords,
		},
		AIFeedback: AIFeedbackBlock{
			Encourage:  llmOut.Encourage,
			ProblemTip: llmOut.ProblemTip,
			HowToFix:   llmOut.HowToFix,
			AiAudioURL: aiAudioURL,
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

type PhonemeIssue struct {
	Phoneme   string
	IssueType string // "missed"(漏读) / "substituted"(替换) / "added"(增读)
}

type SentenceWordIssue struct {
	Word      string
	Score     float64
	IssueType string // "not_pronounced"(根本没读) / "poor_score"(分数低)
}

// LLMCorrectionInput 传给LLM的结构化摘要（不传原始XML）
type LLMCorrectionInput struct {
	PracticeType string // "word" 或 "sentence"
	Content      string // 原文
	OverallScore float64
	ScoreLevel   string // "excellent"/"good"/"fair"/"poor"

	// 单词题用
	WordPhonemeIssues []PhonemeIssue

	// 句子题用
	ProblemWords []SentenceWordIssue
}

// 解析evalResult为LLMCorrectionInput
func buildCorrectionInput(content string, practiceType string, evalResult *domain.EvaluationResult) *LLMCorrectionInput {
	input := &LLMCorrectionInput{
		PracticeType: practiceType,
		Content:      content,
		OverallScore: evalResult.TotalScore,
		ScoreLevel:   toScoreLevel(evalResult.TotalScore),
	}

	if practiceType == "word" {
		input.WordPhonemeIssues = extractPhonemeIssues(evalResult.Words)
	} else {
		input.ProblemWords = extractSentenceWordIssues(evalResult.Words)
	}

	return input
}

func extractPhonemeIssues(words []domain.WordEvaluationResult) []PhonemeIssue {
	var issues []PhonemeIssue
	for _, word := range words {
		for _, p := range word.Phonemes {
			dp := p.DpMessage
			if dp == 0 {
				continue
			}
			issue := PhonemeIssue{Phoneme: p.Phoneme}
			switch dp {
			case 16:
				// 引擎判断该单词或该音素漏读
				issue.IssueType = "missed"
			case 32:
				// 引擎判断该单词或该音素增读
				issue.IssueType = "added"
			default:
				issue.IssueType = "unknown"
			}
			issues = append(issues, issue)
		}
	}
	return issues
}

func extractSentenceWordIssues(words []domain.WordEvaluationResult) []SentenceWordIssue {
	// 过滤掉 sil/fil 等非内容词
	var issues []SentenceWordIssue
	for _, w := range words {
		if w.Word == "sil" || w.Word == "fil" {
			continue
		}
		// dp_message=16
		if w.DpMessage == 16 {
			issues = append(issues, SentenceWordIssue{
				Word:      w.Word,
				Score:     w.Score,
				IssueType: "not_pronounced",
			})
		} else if w.Score < 2.0 && w.Score > 0 {
			// 读了但分数很低
			issues = append(issues, SentenceWordIssue{
				Word:      w.Word,
				Score:     w.Score,
				IssueType: "poor_score",
			})
		}
	}

	// 按严重程度排序，取前2个
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Score < issues[j].Score
	})
	if len(issues) > 2 {
		issues = issues[:2]
	}
	return issues
}

func toScoreLevel(score float64) string {
	switch {
	case score >= 4.0:
		return "excellent"
	case score >= 3.0:
		return "good"
	case score >= 2.0:
		return "fair"
	default:
		return "poor"
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
	Encourage  string `json:"encourage"`   // 鼓励，肯定亮点，1句
	ProblemTip string `json:"problem_tip"` // 指出哪里不对，用孩子听得懂的语言，1句；无问题时为空
	HowToFix   string `json:"how_to_fix"`  // 具体怎么发这个音，身体动作描述，1-2句
	Retry      string `json:"retry"`       // 引导再试一次，1句
}

func (s *PronunciationService) buildFeedbackLLM(
	ctx context.Context,
	rawScore float64,
	stars int,
	input *LLMCorrectionInput,
) llmFeedbackJSON {
	sys := `你是一位儿童英语发音老师，说话简短活泼，语气鼓励正向。只能返回 JSON，不允许返回任何其他内容。`
	usr := buildPromptBody(input, rawScore, stars)
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

func buildPromptBody(input *LLMCorrectionInput, rawScore float64, stars int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("练习内容：\"%s\"（%s）\n", input.Content, practiceTypeCN(input.PracticeType)))
	sb.WriteString(fmt.Sprintf("得分：%.1f/5.0，%d颗星（%s）\n\n", rawScore, stars, input.ScoreLevel))

	// 问题描述部分，单词题和句子题分开写
	if input.PracticeType == "word" {
		if len(input.WordPhonemeIssues) == 0 {
			sb.WriteString("发音情况：所有音素发音正确，整体很好！\n")
		} else {
			sb.WriteString("有问题的音素：\n")
			for _, p := range input.WordPhonemeIssues {
				sb.WriteString(fmt.Sprintf("- /%s/ ：%s\n", p.Phoneme, issueTypeCN(p.IssueType)))
			}
		}
	} else {
		if len(input.ProblemWords) == 0 {
			sb.WriteString("发音情况：整句发音不错，没有明显问题词！\n")
		} else {
			sb.WriteString("需要重点练习的单词：\n")
			for _, w := range input.ProblemWords {
				sb.WriteString(fmt.Sprintf("- %s ：%s\n", w.Word, issueTypeCN(w.IssueType)))
			}
		}
	}

	sb.WriteString(`
现在请你作为儿童英语老师，用最简单活泼的语气，给小朋友反馈。

严格只返回以下JSON格式，不要加任何其他文字，不要解释：

{
  "encourage": "先夸奖孩子，找一个亮点，8-12个字",
  "problem_tip": "用孩子能听懂的话说哪里不对，最多12个字。没有问题就填空字符串",
  "how_to_fix": "教孩子嘴巴、舌头怎么动，动作要清楚具体，10-18个字。没有问题就填空字符串",
  "retry": "鼓励孩子再读一次，活泼一点，6-10个字"
}

例子1（单词 dog，oo音不好）：
{
  "encourage": "哇，dog读得很有精神！",
  "problem_tip": "中间的 oo 音有点太紧了",
  "how_to_fix": "嘴巴张大一点，像医生说 ah—— 那样",
  "retry": "来，我们再试一次！"
}

例子2（句子整体很好）：
{
  "encourage": "你读得真棒！声音好清楚！",
  "problem_tip": "",
  "how_to_fix": "",
  "retry": "再读一遍给老师听听～"
}
`)

	return sb.String()
}

func practiceTypeCN(t string) string {
	if t == "word" {
		return "单词朗读"
	}
	return "句子朗读"
}

func issueTypeCN(t string) string {
	switch t {
	case "missed":
		return "这个音漏掉了，没有读出来"
	case "substituted":
		return "这个音读错了"
	case "added":
		return "多读了一个不该有的音"
	case "not_pronounced":
		return "这个词没被读出来"
	case "poor_score":
		return "这个词发音得分很低"
	default:
		return "发音有问题"
	}
}

func fallbackFeedback(stars int) llmFeedbackJSON {
	if stars >= 4 {
		return llmFeedbackJSON{
			Encourage:  "读得很棒！⭐",
			ProblemTip: "",
			HowToFix:   "",
			Retry:      "再来一遍，更棒！",
		}
	}
	return llmFeedbackJSON{
		Encourage:  "敢开口就是进步！💪",
		ProblemTip: "跟着示范再听一遍。",
		HowToFix:   "慢慢来，一个音一个音练。",
		Retry:      "再试一次，你可以的！",
	}
}

func clampFloat64(v, min, max float64) float64 {
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

// scoreBucket 将 rawScore 按 0.5 分一档分桶，用于缓存 key 生成
// 例如：3.7 → "3.5"，4.2 → "4.0"，5.0 → "5.0"
func scoreBucket(score float64) string {
	bucketed := math.Floor(score*2) / 2
	return fmt.Sprintf("%.1f", bucketed)
}

// correctionFingerprint 根据 correctionInput 生成问题指纹
// 单词题：音素列表的 MD5，如  "g:missed,th:added" 得到相同指纹
// 句子题：问题词+类型的 MD5，如 "beginning:not_pronounced,breath:poor_score"
func correctionFingerprint(input *LLMCorrectionInput) string {
	var parts []string

	if input.PracticeType == "word" {
		for _, p := range input.WordPhonemeIssues {
			parts = append(parts, p.Phoneme+":"+p.IssueType)
		}
	} else {
		for _, w := range input.ProblemWords {
			parts = append(parts, w.Word+":"+w.IssueType)
		}
	}

	// 排序保证相同问题不同顺序得到相同 key
	sort.Strings(parts)
	raw := strings.Join(parts, ",")
	if raw == "" {
		raw = "no_problem"
	}
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// buildFeedbackLLMWithCache 带 Redis 缓存的 LLM 发音反馈生成
// Key 格式：pron:llm:feedback:{unitID}:{itemID}:{practiceType}:{scoreBucket}:{correctionFingerprint}
// TTL：7天（相同题目、相同分档、相同错误词，反馈固定）
func (s *PronunciationService) buildFeedbackLLMWithCache(
	ctx context.Context,
	unitID string,
	itemID int,
	content string,
	practiceType string,
	rawScore float64,
	stars int,
	correctionInput *LLMCorrectionInput,
) llmFeedbackJSON {
	bucket := scoreBucket(rawScore)
	fingerprint := correctionFingerprint(correctionInput)
	key := fmt.Sprintf("pron:llm:feedback:%s:%d:%s:%s:%s",
		unitID, itemID, practiceType, bucket, fingerprint)

	if val, err := s.cache.Get(ctx, key); err == nil {
		var cached llmFeedbackJSON
		if json.Unmarshal([]byte(val), &cached) == nil {
			s.logger.Info("pron llm feedback cache hit", slog.String("key", key))
			return cached
		}
	}

	result := s.buildFeedbackLLM(ctx, rawScore, stars, correctionInput)

	if b, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, key, string(b), 7*24*time.Hour); err != nil {
			s.logger.Warn("pron llm feedback cache set failed", slog.String("error", err.Error()))
		}
	}
	return result
}

// synthPronAudioWithCache 带 Redis 缓存的 TTS 合成（发音纠正模块专用）
// Key 格式：pron:tts:{ttsText 的 MD5}
// TTL：30天（相同文本合成结果永不变）
// 缓存存储 OSS URL，不存音频字节
func (s *PronunciationService) synthPronAudioWithCache(
	ctx context.Context,
	sessionID string,
	ttsText string,
) string {
	if strings.TrimSpace(ttsText) == "" {
		return ""
	}

	cacheKey := fmt.Sprintf("pron:tts:%s", md5str(ttsText))

	if val, err := s.cache.Get(ctx, cacheKey); err == nil && val != "" {
		s.logger.Info("pron tts cache hit", slog.String("key", cacheKey))
		return val
	}

	audio, err := s.ttsProvider.Synthesize(ctx, ttsText, nil)
	if err != nil {
		s.logger.Warn("pronunciation v2 TTS failed", slog.String("error", err.Error()))
		return ""
	}

	aiKey := fmt.Sprintf("pronunciation/%s/ai_%s.wav", sessionID, uuid.New())
	url, upErr := s.ossProvider.UploadAudio(ctx, aiKey, audio)
	if upErr != nil {
		s.logger.Warn("pronunciation v2 AI audio upload failed", slog.String("error", upErr.Error()))
		return ""
	}

	if err := s.cache.Set(ctx, cacheKey, url, 30*24*time.Hour); err != nil {
		s.logger.Warn("pron tts cache set failed", slog.String("error", err.Error()))
	}
	return url
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
