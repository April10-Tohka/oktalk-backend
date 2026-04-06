package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/repository"
)

// HTTPError 业务层可映射到 HTTP 状态码的错误
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

func errHTTP(status int, msg string) error {
	return &HTTPError{Status: status, Message: msg}
}

// SceneService 场景引导对话
type SceneService struct {
	sceneLoader *config.SceneLoader
	sessionRepo repository.SceneSessionRepository
	messageRepo repository.SceneMessageRepository
	asrProvider domain.ASRProvider
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider
	ossProvider domain.OSSProvider
	cache       repository.SceneCacheRepository // Redis 缓存
	logger      *slog.Logger
}

// NewSceneService 构造函数
func NewSceneService(
	sceneLoader *config.SceneLoader,
	sessionRepo repository.SceneSessionRepository,
	messageRepo repository.SceneMessageRepository,
	asr domain.ASRProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	logger *slog.Logger,
	cache repository.SceneCacheRepository,
) *SceneService {
	return &SceneService{
		sceneLoader: sceneLoader,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		asrProvider: asr,
		llmProvider: llm,
		ttsProvider: tts,
		ossProvider: oss,
		cache:       cache,
		logger:      logger,
	}
}

// --- GetSceneList ---

// SceneListItem 场景列表项
type SceneListItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CoverEmoji  string `json:"cover_emoji"`
	TotalSteps  int    `json:"total_steps"`
}

// GetSceneList 获取全部场景
func (s *SceneService) GetSceneList(ctx context.Context) []SceneListItem {
	all := s.sceneLoader.GetAll()
	out := make([]SceneListItem, 0, len(all))
	for _, sc := range all {
		out = append(out, SceneListItem{
			ID:          sc.ID,
			Title:       sc.Title,
			Description: sc.Description,
			CoverEmoji:  sc.CoverEmoji,
			TotalSteps:  len(sc.Steps),
		})
	}
	return out
}

// --- StartSession ---

// SceneStartSessionRequest 开始场景会话请求
type SceneStartSessionRequest struct {
	UserID  string
	SceneID string
}

// SceneStartSessionResponse 开始场景会话响应
type SceneStartSessionResponse struct {
	SessionID         string `json:"session_id"`
	IsResumed         bool   `json:"is_resumed"`
	CurrentStep       int    `json:"current_step"`
	Question          string `json:"question"`
	QuestionAudioURL  string `json:"question_audio_url"`
	QuestionAudioText string `json:"question_audio_text"`
}

// StartSession 开始或续学场景
func (s *SceneService) StartSession(ctx context.Context, req *SceneStartSessionRequest) (*SceneStartSessionResponse, error) {
	if _, ok := s.sceneLoader.GetByID(req.SceneID); !ok {
		return nil, errHTTP(400, "场景不存在")
	}

	var (
		sess        *model.SceneSession
		isResumed   bool
		currentStep int
		step        *config.SceneStep
		sessionID   string
	)

	existing, err := s.sessionRepo.FindActiveByUserAndScene(ctx, req.UserID, req.SceneID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		isResumed = true
		sess = existing
		sessionID = existing.ID
		currentStep = existing.CurrentStep
		st, ok := s.sceneLoader.GetStep(req.SceneID, currentStep)
		if !ok {
			return nil, errHTTP(500, "场景步骤配置异常")
		}
		step = st
	} else {
		sessionID = uuid.New()
		currentStep = 1
		st, ok := s.sceneLoader.GetStep(req.SceneID, 1)
		if !ok {
			return nil, errHTTP(500, "场景步骤配置异常")
		}
		step = st
		sess = &model.SceneSession{
			ID:          sessionID,
			UserID:      req.UserID,
			SceneID:     req.SceneID,
			CurrentStep: 1,
			Status:      "active",
		}
		if err := s.sessionRepo.Create(ctx, sess); err != nil {
			return nil, err
		}
	}

	qURL := s.synthQuestionAudio(ctx, sess.SceneID, currentStep, step.QuestionAudioText)

	return &SceneStartSessionResponse{
		SessionID:         sessionID,
		IsResumed:         isResumed,
		CurrentStep:       currentStep,
		Question:          step.Question,
		QuestionAudioURL:  qURL,
		QuestionAudioText: step.QuestionAudioText,
	}, nil
}

// synthQuestionAudio 带缓存的场景问题音频合成
// Key 格式：scene:tts:question:{sceneID}:{stepID}
// TTL：30天（场景配置固定，音频永不变）
func (s *SceneService) synthQuestionAudio(ctx context.Context, sceneID string, stepID int, text string) string {
	cacheKey := fmt.Sprintf("scene:tts:question:%s:%d", sceneID, stepID)
	return s.synthAudioWithCache(ctx, text, cacheKey, 30*24*time.Hour, func() string {
		return fmt.Sprintf("scene/questions/%s_%d.wav", sceneID, stepID)
	})
}

// --- SubmitAnswer ---

// SubmitAnswerRequest 提交回答
type SubmitAnswerRequest struct {
	UserID    string
	SessionID string
	StepID    int
	AudioData []byte
	AudioType string // wav / mp3 / pcm
}

// SubmitAnswerResponse 提交回答响应
type SubmitAnswerResponse struct {
	UserText             string `json:"user_text"`
	UserAudioURL         string `json:"user_audio_url"`
	MatchResult          string `json:"match_result"`
	AIReplyText          string `json:"ai_reply_text"`
	AIAudioURL           string `json:"ai_audio_url"`
	ShouldAdvance        bool   `json:"step_advanced"`
	SceneCompleted       bool   `json:"scene_completed"`
	CurrentStep          int    `json:"current_step"`
	NextQuestion         string `json:"next_question,omitempty"`
	NextQuestionAudioURL string `json:"next_question_audio_url,omitempty"`
}

// SubmitAnswer 处理用户语音回答
func (s *SceneService) SubmitAnswer(ctx context.Context, req *SubmitAnswerRequest) (*SubmitAnswerResponse, error) {
	if req.SessionID == "" || req.StepID <= 0 {
		return nil, errHTTP(400, "参数无效")
	}
	if len(req.AudioData) == 0 {
		return nil, errHTTP(400, "音频为空，请重新录音")
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
	if sess.Status != "active" {
		return nil, errHTTP(400, "场景已完成")
	}
	if sess.CurrentStep != req.StepID {
		return nil, errHTTP(400, "步骤不匹配")
	}

	scene, ok := s.sceneLoader.GetByID(sess.SceneID)
	if !ok {
		return nil, errHTTP(500, "场景配置异常")
	}
	step, ok := s.sceneLoader.GetStep(sess.SceneID, req.StepID)
	if !ok {
		return nil, errHTTP(500, "数据异常")
	}

	prevCount, err := s.messageRepo.CountAttempts(ctx, req.SessionID, req.StepID)
	if err != nil {
		return nil, err
	}
	attempt := prevCount + 1

	asrText, err := s.asrProvider.RecognizeAudio(ctx, req.AudioData)
	if err != nil {
		s.logger.Error("scene ASR failed", slog.String("error", err.Error()))
		return nil, errHTTP(500, "语音识别失败")
	}
	userText := strings.TrimSpace(asrText)
	if userText == "" {
		return nil, errHTTP(400, "音频为空，请重新录音")
	}

	userAudioURL := ""
	userAudioKey := fmt.Sprintf("scene/%s/user_%s.wav", req.SessionID, uuid.New())
	go func(data []byte, key string) {
		ctx2, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if url, err := s.ossProvider.UploadAudio(ctx2, key, data); err != nil {
			s.logger.Warn("scene user audio upload failed", slog.String("error", err.Error()))
		} else {
			userAudioURL = url
		}

	}(append([]byte(nil), req.AudioData...), userAudioKey)

	userTextLower := strings.ToLower(userText)
	matchResult := "fail"
	llmStatus := ""

	for _, e := range step.Expected {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" && strings.Contains(userTextLower, e) {
			matchResult = "rule_pass"
			break
		}
	}

	if matchResult != "rule_pass" {
		semOK, st := s.llmSemanticMatchWithCache(ctx, sess.SceneID, req.StepID, scene.Title, step.Question, step.Expected, userText)
		llmStatus = st
		if semOK {
			matchResult = "llm_pass"
		} else {
			matchResult = "fail"
		}
	}

	reply := s.buildReplyJSONWithCache(ctx, scene, step, sess.SceneID, req.StepID, userText, matchResult, attempt)

	aiAudioURL := ""
	if reply.Reply != "" {
		cacheKey := fmt.Sprintf("scene:tts:%s", md5str(reply.Reply))
		aiAudioURL = s.synthAudioWithCache(ctx, reply.Reply, cacheKey, 7*24*time.Hour, func() string {
			return fmt.Sprintf("scene/%s/ai_%s.wav", req.SessionID, uuid.New())
		})
	}

	shouldAdvance := false
	switch matchResult {
	case "rule_pass", "llm_pass":
		shouldAdvance = true
	case "fail":
		if attempt >= 2 {
			shouldAdvance = true
		}
	}

	totalSteps := len(scene.Steps)
	isLastStep := req.StepID == totalSteps
	sceneCompleted := false
	nextQ := ""
	nextAudioURL := ""
	newCurrent := sess.CurrentStep

	if shouldAdvance {
		if isLastStep {
			if err := s.sessionRepo.UpdateStatus(ctx, req.SessionID, "completed"); err != nil {
				return nil, err
			}
			sceneCompleted = true
			newCurrent = req.StepID
		} else {
			nextID := req.StepID + 1
			if err := s.sessionRepo.UpdateCurrentStep(ctx, req.SessionID, nextID); err != nil {
				return nil, err
			}
			newCurrent = nextID
			if nst, ok := s.sceneLoader.GetStep(sess.SceneID, nextID); ok {
				nextQ = nst.Question
				nextAudioURL = s.synthQuestionAudio(ctx, sess.SceneID, nextID, nst.QuestionAudioText)
			}
		}
	} else {
		newCurrent = sess.CurrentStep
		if nst, ok := s.sceneLoader.GetStep(sess.SceneID, newCurrent); ok {
			nextQ = nst.Question
			nextAudioURL = s.synthQuestionAudio(ctx, sess.SceneID, newCurrent, nst.QuestionAudioText)
		}
	}

	msg := &model.SceneMessage{
		ID:           uuid.New(),
		SessionID:    req.SessionID,
		UserID:       req.UserID,
		SceneID:      sess.SceneID,
		StepID:       req.StepID,
		Attempt:      attempt,
		UserText:     userText,
		UserAudioURL: "",
		MatchResult:  matchResult,
		AIReplyText:  reply.Reply,
		AIAudioURL:   aiAudioURL,
		LLMStatus:    llmStatus,
		StepAdvanced: shouldAdvance,
	}
	msg.UserAudioURL = s.ossProvider.GetPublicURL(userAudioKey)

	go func(m *model.SceneMessage) {
		ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.messageRepo.Create(ctx2, m); err != nil {
			s.logger.Warn("scene message persist failed", slog.String("error", err.Error()))
		}
	}(msg)

	return &SubmitAnswerResponse{
		UserText:             userText,
		UserAudioURL:         userAudioURL,
		MatchResult:          matchResult,
		AIReplyText:          reply.Reply,
		AIAudioURL:           aiAudioURL,
		ShouldAdvance:        shouldAdvance,
		SceneCompleted:       sceneCompleted,
		CurrentStep:          newCurrent,
		NextQuestion:         nextQ,
		NextQuestionAudioURL: nextAudioURL,
	}, nil
}

type llmReplyParsed struct {
	Status string `json:"status"`
	Reply  string `json:"reply"`
	Hint   string `json:"hint,omitempty"`
	Expand string `json:"expand,omitempty"`
}

func (s *SceneService) buildReplyJSON(ctx context.Context, scene *config.SceneConfig, step *config.SceneStep, userText, matchResult string, attempt int) llmReplyParsed {
	expectedStr := ""
	if len(step.Expected) > 0 {
		expectedStr = strings.Join(step.Expected, ", ")
	}

	var sys, usr string
	if matchResult == "rule_pass" || matchResult == "llm_pass" {
		sys = `你是一位友好的儿童英语老师，说话简短活泼，多用 emoji。
只能返回 JSON，不允许返回任何其他内容。`
		usr = fmt.Sprintf(`场景：%s
小朋友正确回答了："%s"
期望的表达是：%s

请生成简短的鼓励回复（英文，20词以内），可选加入简单拓展。
只返回如下 JSON：
{"status":"pass","reply":"...","expand":"..."}
expand 字段可选，没有则省略。`, scene.Title, userText, expectedStr)
	} else if matchResult == "fail" {
		if attempt == 1 {
			sys = `你是一位友好的儿童英语老师，说话简短活泼，多用 emoji。
只能返回 JSON，不允许返回任何其他内容。`
			usr = fmt.Sprintf(`场景：%s
问题："%s"
小朋友回答了："%s"，但没有包含正确表达。
这是第一次尝试，请鼓励小朋友再试一次，给出提示但不要直接给出答案。
只返回如下 JSON：
{"status":"fail","reply":"...","hint":"..."}`, scene.Title, step.Question, userText)
		} else {
			answerDemo := expectedStr
			if step.Teach != "" {
				answerDemo = step.Teach
			}
			hintWord := expectedStr
			if len(step.Expected) > 0 {
				hintWord = step.Expected[0]
			}
			sys = `你是一位友好的儿童英语老师，说话简短活泼，多用 emoji。
只能返回 JSON，不允许返回任何其他内容。`
			usr = fmt.Sprintf(`场景：%s
问题："%s"
小朋友尝试了两次都没答对，正确答案是："%s"
请温柔地示范正确答案并鼓励继续。
只返回如下 JSON：
{"status":"fail","reply":"...","hint":"Say: %s"}`, scene.Title, step.Question, answerDemo, hintWord)
		}
	} else {
		return llmReplyParsed{Status: "pass", Reply: "Great job! Let's continue!"}
	}

	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	if err != nil {
		s.logger.Warn("scene LLM reply failed", slog.String("error", err.Error()))
		return fallbackReply(matchResult, step)
	}
	js := extractJSONObject(raw)
	var out llmReplyParsed
	if err := json.Unmarshal([]byte(js), &out); err != nil || out.Reply == "" {
		s.logger.Warn("scene LLM reply JSON parse failed", slog.String("raw", raw))
		return fallbackReply(matchResult, step)
	}
	return out
}

func fallbackReply(matchResult string, step *config.SceneStep) llmReplyParsed {
	if matchResult == "rule_pass" || matchResult == "llm_pass" {
		return llmReplyParsed{Status: "pass", Reply: "Great job! Let's continue!"}
	}
	hint := "Say: something"
	if len(step.Expected) > 0 {
		hint = "Say: " + step.Expected[0]
	}
	return llmReplyParsed{Status: "fail", Reply: "Good try! Let's try again!", Hint: hint}
}

type semanticLLMResp struct {
	SemanticMatch bool `json:"semantic_match"`
}

func (s *SceneService) llmSemanticMatch(ctx context.Context, sceneTitle, question string, expected []string, userText string) (bool, string) {
	sys := `你是一个英语教学判断器，只能返回 JSON，不允许返回任何其他内容。`
	usr := fmt.Sprintf(`场景：%s
当前问题：%s
期望关键词：%s
用户回答：%s

判断用户的回答语义上是否包含了期望的表达。
只返回如下格式的 JSON，不要有任何其他内容：
{"semantic_match": true} 或 {"semantic_match": false}`, sceneTitle, question, strings.Join(expected, ", "), userText)

	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	if err != nil {
		s.logger.Warn("scene LLM semantic failed", slog.String("error", err.Error()))
		return false, "fail"
	}
	js := extractJSONObject(raw)
	var out semanticLLMResp
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		s.logger.Warn("scene semantic JSON parse failed", slog.String("raw", raw))
		return false, "fail"
	}
	if out.SemanticMatch {
		return true, "pass"
	}
	return false, "fail"
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "{"); idx >= 0 {
		if j := strings.LastIndex(s, "}"); j > idx {
			return s[idx : j+1]
		}
	}
	return s
}

// normalizeText 标准化用户文本：转小写、去首尾空格、合并连续空格
// 用于缓存 key 生成，覆盖 ASR 微小识别差异
func normalizeText(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	return strings.Join(strings.Fields(t), " ")
}

// md5str 返回字符串的 MD5 十六进制摘要，用于缓存 key
func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// synthAudioWithCache 通用带缓存 TTS 合成
// 缓存存储的是 OSS URL（不存音频字节，节省 Redis 内存）
// uploadKeyFn 用于生成 OSS 上传路径
func (s *SceneService) synthAudioWithCache(
	ctx context.Context,
	text string,
	cacheKey string,
	ttl time.Duration,
	uploadKeyFn func() string,
) string {
	if text == "" {
		return ""
	}

	if val, err := s.cache.Get(ctx, cacheKey); err == nil && val != "" {
		s.logger.Info("scene tts cache hit", slog.String("key", cacheKey))
		return val
	}

	opts := domain.DefaultSynthesizeOptions()
	opts.Format = "wav"
	audio, err := s.ttsProvider.Synthesize(ctx, text, opts)
	if err != nil {
		s.logger.Warn("scene TTS synthesize failed", slog.String("error", err.Error()))
		return ""
	}

	url, err := s.ossProvider.UploadAudio(ctx, uploadKeyFn(), audio)
	if err != nil {
		s.logger.Warn("scene OSS upload failed", slog.String("error", err.Error()))
		return ""
	}

	if err := s.cache.Set(ctx, cacheKey, url, ttl); err != nil {
		s.logger.Warn("scene tts cache set failed", slog.String("error", err.Error()))
	}
	return url
}

// llmSemanticMatchWithCache 带 Redis 缓存的语义判断
// Key 格式：scene:llm:semantic:{sceneID}:{stepID}:{normalizedText的MD5}
// TTL：7天（相同表达语义固定）
func (s *SceneService) llmSemanticMatchWithCache(
	ctx context.Context,
	sceneID string,
	stepID int,
	sceneTitle, question string,
	expected []string,
	userText string,
) (bool, string) {
	norm := normalizeText(userText)
	key := fmt.Sprintf("scene:llm:semantic:%s:%d:%s", sceneID, stepID, md5str(norm))

	if val, err := s.cache.Get(ctx, key); err == nil {
		s.logger.Info("scene llm semantic cache hit", slog.String("key", key))
		matched := val == "true"
		status := "fail"
		if matched {
			status = "pass"
		}
		return matched, status
	}

	ok, status := s.llmSemanticMatch(ctx, sceneTitle, question, expected, userText)

	cacheVal := "false"
	if ok {
		cacheVal = "true"
	}
	if err := s.cache.Set(ctx, key, cacheVal, 7*24*time.Hour); err != nil {
		s.logger.Warn("scene llm semantic cache set failed", slog.String("error", err.Error()))
	}
	return ok, status
}

// buildReplyJSONWithCache 带 Redis 缓存的 LLM 回复生成
// Key 格式：scene:llm:reply:{sceneID}:{stepID}:{matchResult}:{attempt}:{normalizedText的MD5}
// TTL：7天
func (s *SceneService) buildReplyJSONWithCache(
	ctx context.Context,
	scene *config.SceneConfig,
	step *config.SceneStep,
	sceneID string,
	stepID int,
	userText, matchResult string,
	attempt int,
) llmReplyParsed {
	norm := normalizeText(userText)
	key := fmt.Sprintf("scene:llm:reply:%s:%d:%s:%d:%s",
		sceneID, stepID, matchResult, attempt, md5str(norm))

	if val, err := s.cache.Get(ctx, key); err == nil {
		var cached llmReplyParsed
		if json.Unmarshal([]byte(val), &cached) == nil {
			s.logger.Info("scene llm reply cache hit", slog.String("key", key))
			return cached
		}
	}

	result := s.buildReplyJSON(ctx, scene, step, userText, matchResult, attempt)

	if b, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, key, string(b), 7*24*time.Hour); err != nil {
			s.logger.Warn("scene llm reply cache set failed", slog.String("error", err.Error()))
		}
	}
	return result
}

// --- GetSummary ---

// SceneSummaryResponse 场景总结
type SceneSummaryResponse struct {
	SummaryIntro string   `json:"summary_intro"`
	SummaryItems []string `json:"summary_items"`
	PassedSteps  int      `json:"passed_steps"`
	TotalSteps   int      `json:"total_steps"`
	PassRate     int      `json:"pass_rate"`
}

// GetSummary 获取已完成场景的总结
func (s *SceneService) GetSummary(ctx context.Context, userID, sessionID string) (*SceneSummaryResponse, error) {
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
	if sess.Status != "completed" {
		return nil, errHTTP(400, "场景未完成，无法查看总结")
	}

	scene, ok := s.sceneLoader.GetByID(sess.SceneID)
	if !ok {
		return nil, errHTTP(500, "场景配置异常")
	}

	passed, err := s.messageRepo.CountPassedSteps(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	total := len(scene.Steps)
	passRate := 0
	if total > 0 {
		passRate = passed * 100 / total
	}

	return &SceneSummaryResponse{
		SummaryIntro: scene.SummaryIntro,
		SummaryItems: scene.SummaryItems,
		PassedSteps:  passed,
		TotalSteps:   total,
		PassRate:     passRate,
	}, nil
}

// --- GetHistory ---

// GetHistoryResponse 历史记录
type GetHistoryResponse struct {
	Messages []*model.SceneMessage `json:"messages"`
}

// GetHistory 获取会话消息历史
func (s *SceneService) GetHistory(ctx context.Context, userID, sessionID string) (*GetHistoryResponse, error) {
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

	list, err := s.messageRepo.ListBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &GetHistoryResponse{Messages: list}, nil
}
