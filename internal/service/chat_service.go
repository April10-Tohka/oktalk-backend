// Package service 提供 AI 语音对话业务逻辑
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
	domainLimiter "pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/infrastructure/vad/vadpb"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/worker"

	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ===== 请求结构 =====

// ChatMVPRequest MVP 同步语音对话请求
type ChatMVPRequest struct {
	AudioData        []byte
	AudioType        string // wav / mp3
	ConversationType string // free_talk / question_answer
	DifficultyLevel  string // beginner / intermediate / advanced
	UserID           string
	ConversationID   string
}

// SubmitChatRequest 异步语音对话提交请求
type SubmitChatRequest struct {
	AudioData        []byte
	AudioType        string
	ConversationID   string // 由 /session/start 预先创建
	ConversationType string // free_talk / question_answer
	DifficultyLevel  string // beginner / intermediate / advanced
	Topic            string
	UserID           string
}

// ChatHistoryRequest 对话历史查询请求
type ChatHistoryRequest struct {
	SessionID string
	Page      int
	PageSize  int
	Order     string // asc / desc
	UserID    string
}

// SubmitFeedbackRequest 对话反馈提交请求
type SubmitFeedbackRequest struct {
	TaskID    string
	SessionID string
	Turn      int
	Rating    int
	Comment   string
	Helpful   bool
	UserID    string
}

type StartSessionRequest struct {
	ConversationType string
	DifficultyLevel  string
	Topic            string
	UserID           string
}

type ValidateFreetalkRequest struct {
	ConversationID string
	UserID         string
}

type HandleFreetalkRequest struct {
	Conn           *websocket.Conn
	UserID         string
	ConversationID string
}

// ===== 响应结构 =====

type StartSessionResponse struct {
	SessionId        string `json:"session_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
	DifficultyLevel  string `json:"difficulty_level"`
	Topic            string `json:"topic"`
	Status           string `json:"status"`
}

// ChatResultResponse 语音对话处理结果
type ChatResultResponse struct {
	TaskID       string      `json:"task_id"`
	Status       string      `json:"status"`
	CurrentStage string      `json:"current_stage,omitempty"`
	Message      string      `json:"message,omitempty"`
	SessionID    string      `json:"session_id,omitempty"`
	UserInput    *AudioInput `json:"user_input,omitempty"`
	AIResponse   *AudioReply `json:"ai_response,omitempty"`
	CreatedAt    int64       `json:"created_at,omitempty"`
	FeedbackURL  string      `json:"feedback_url,omitempty"`
	ErrorStage   string      `json:"error_stage,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// AudioInput 用户语音识别结果
type AudioInput struct {
	Text       string `json:"text"`
	DurationMs int    `json:"duration_ms"`
}

// AudioReply AI 回复（文本 + 音频）
type AudioReply struct {
	Text       string `json:"text"`
	AudioURL   string `json:"audio_url"`
	DurationMs int    `json:"duration_ms"`
}

// ConversationTurn 单轮对话记录
type ConversationTurn struct {
	Turn         int    `json:"turn"`
	UserText     string `json:"user_text"`
	UserAudioURL string `json:"user_audio_url"`
	AIText       string `json:"ai_text"`
	AIAudioURL   string `json:"ai_audio_url"`
	CreatedAt    string `json:"created_at"`
}

// SessionSummary 会话摘要
type SessionSummary struct {
	SessionID         string `json:"session_id"`
	CreatedAt         string `json:"created_at"`
	LastMessage       string `json:"last_message"`
	MessageCount      int    `json:"message_count"`
	LastInteractionAt string `json:"last_interaction_at"`
}

// ===== Service 接口 =====

// ChatService AI 语音对话业务接口
type ChatService interface {
	// ChatMVP 同步语音对话 MVP（ASR → LLM → TTS）
	ChatMVP(ctx context.Context, req *ChatMVPRequest) ([]byte, error)

	// StartSession 创建新的对话会话
	StartSession(ctx context.Context, req *StartSessionRequest) (*StartSessionResponse, error)

	// SubmitChat 提交异步语音对话任务
	SubmitChat(ctx context.Context, req *SubmitChatRequest) (taskID string, err error)

	// GetChatResult 查询异步语音对话处理结果
	GetChatResult(ctx context.Context, taskID string) (*ChatResultResponse, error)

	// GetChatHistory 获取指定会话的对话历史
	GetChatHistory(ctx context.Context, req *ChatHistoryRequest) ([]*ConversationTurn, int64, error)

	// DeleteSession 删除对话会话及其所有消息
	DeleteSession(ctx context.Context, sessionID, userID string) (int64, error)

	// GetSessions 获取用户的会话列表
	GetSessions(ctx context.Context, userID string, page, pageSize int) ([]*SessionSummary, int64, error)

	// SubmitChatFeedback 提交对话反馈
	SubmitChatFeedback(ctx context.Context, req *SubmitFeedbackRequest) error

	// ValidateFreetalk 验证 free talk 模式语音对话请求
	ValidateFreetalk(ctx context.Context, req *ValidateFreetalkRequest) error

	// HandleFreetalk 处理 free talk 模式语音对话请求
	HandleFreetalk(ctx context.Context, req *HandleFreetalkRequest) error

	// GetFreetalkSummary 获取 free talk 模式语音对话摘要
	GetFreetalkSummary(ctx context.Context, sessionID, userID string) (*SessionSummaryResponse, error)
}

// ===== Service 实现 =====

// chatServiceImpl Chat Service 实现
type chatServiceImpl struct {
	conversationRepo db.VoiceConversationRepository
	messageRepo      db.ConversationMessageRepository
	asrProvider      domain.ASRProvider
	llmProvider      domain.LLMProvider
	ttsProvider      domain.TTSProvider
	ossProvider      domain.OSSProvider
	// vadClient VAD gRPC 客户端，由 NewChatService 注入；HandleFreetalk 透传给 Session
	vadClient vadpb.VADServiceClient
	// pronunciationService / reportService 注入给 FreeTalk Session，供 Agent 工具使用（可为 nil）
	pronunciationService *PronunciationService
	reportService        ReportService
	taskCache            *cache.TaskCache
	chatCache            *cache.ChatCache
	workerManager        *worker.Manager
	rateLimitFactory     domainLimiter.SceneLimiterFactory
	logger               *slog.Logger
}

// NewChatService 创建 ChatService
func NewChatService(
	repos *db.Repositories,
	asr domain.ASRProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	vadClient vadpb.VADServiceClient,
	taskCache *cache.TaskCache,
	chatCache *cache.ChatCache,
	workerMgr *worker.Manager,
	rlFactory domainLimiter.SceneLimiterFactory,
	logger *slog.Logger,
	pronService *PronunciationService,
	reportService ReportService,
) ChatService {
	var conversationRepo db.VoiceConversationRepository
	var messageRepo db.ConversationMessageRepository
	if repos != nil {
		conversationRepo = repos.VoiceConversation
		messageRepo = repos.ConversationMessage
	}
	return &chatServiceImpl{
		conversationRepo:     conversationRepo,
		messageRepo:          messageRepo,
		asrProvider:          asr,
		llmProvider:          llm,
		ttsProvider:          tts,
		ossProvider:          oss,
		vadClient:            vadClient,
		pronunciationService: pronService,
		reportService:        reportService,
		taskCache:            taskCache,
		chatCache:            chatCache,
		workerManager:        workerMgr,
		rateLimitFactory:     rlFactory,
		logger:               logger,
	}
}

func (s *chatServiceImpl) StartSession(ctx context.Context, req *StartSessionRequest) (*StartSessionResponse, error) {
	if req == nil {
		err := errors.New("start session request is nil")
		logger.ErrorContext(ctx, "start session request invalid", "error", err)
		return nil, err
	}
	if req.UserID == "" {
		err := errors.New("user id is empty")
		logger.ErrorContext(ctx, "start session user id missing", "error", err)
		return nil, err
	}
	if s.conversationRepo == nil {
		err := errors.New("conversation repository not initialized")
		logger.ErrorContext(ctx, "start session repository missing", "error", err)
		return nil, err
	}

	sessionId := uuid.New()
	conversation := &model.VoiceConversation{
		ID:              sessionId,
		UserID:          req.UserID,
		MessageCount:    0,
		DurationSeconds: 0,
		Status:          "active",
	}
	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		logger.ErrorContext(ctx, "start session create conversation failed", "error", err)
		return nil, err
	}
	freeTalkSession := &model.FreeTalkSession{
		ID:        uuid.New(),
		SessionID: sessionId,
		UserID:    req.UserID,
		StartedAt: time.Now(),
	}
	dsn := "tohka:Tohka10xiang@tcp(8.155.145.36:3306)/oktalk?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.ErrorContext(ctx, "start session failed to connect database", "error", err)
		return nil, err
	}
	db.Create(freeTalkSession)

	return &StartSessionResponse{
		SessionId: sessionId,
		Status:    "active",
	}, nil
}

func (s *chatServiceImpl) ChatMVP(ctx context.Context, req *ChatMVPRequest) ([]byte, error) {
	// 步骤 1：基础校验
	if req == nil {
		err := errors.New("chat mvp request is nil")
		logger.ErrorContext(ctx, "chat mvp request invalid", "error", err)
		return nil, err
	}
	if len(req.AudioData) == 0 {
		err := errors.New("audio data is empty")
		logger.ErrorContext(ctx, "chat mvp audio empty", "error", err)
		return nil, err
	}
	if req.UserID == "" {
		err := errors.New("user id is empty")
		logger.ErrorContext(ctx, "chat mvp user id missing", "error", err)
		return nil, err
	}
	if s.asrProvider == nil || s.llmProvider == nil || s.ttsProvider == nil {
		err := errors.New("chat mvp provider not initialized")
		logger.ErrorContext(ctx, "chat mvp provider missing", "error", err)
		return nil, err
	}

	audioType := strings.ToLower(strings.TrimSpace(req.AudioType))
	if audioType == "" {
		audioType = "wav"
	}
	conversationType := strings.TrimSpace(req.ConversationType)
	if conversationType == "" {
		conversationType = "free_talk"
	}
	difficultyLevel := strings.TrimSpace(req.DifficultyLevel)
	if difficultyLevel == "" {
		difficultyLevel = "beginner"
	}

	// 步骤 2：ASR 识别
	asrResult, err := s.asrProvider.RecognizeAudio(ctx, req.AudioData)
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp asr failed", "error", err)
		return nil, err
	}
	userText := strings.TrimSpace(asrResult)
	if userText == "" {
		err := errors.New("asr result is empty")
		logger.ErrorContext(ctx, "chat mvp asr empty", "error", err)
		return nil, err
	}

	// 步骤 3：LLM 生成回复
	systemPrompt := `
You are a friendly English teacher for Chinese kids (6-12 years old) learning English.

CORE RULES:
1. Response length: Maximum 25 words (2 sentences)
2. Vocabulary: Use only simple, common words (like: cat, happy, play, eat, go)
3. Primary language: English
4. When to use Chinese: Only for difficult grammar explanations (use 【】brackets)
5. Always be encouraging and positive

Response Pattern:
- Child speaks English → Reply in simple English + praise
- Child speaks Chinese → Gently prompt in English: "Let's try English! You can say..."
- Child makes mistakes → Don't correct directly, just model the right form

Examples:
Child: "I go school yesterday" 
You: "Great! I went to school yesterday too. What did you do there?"

Child: "这个怎么说？"
You: "Let's say it in English! You can ask: How do you say this?"

Child: "I'm happy!"
You: "Wonderful! I'm happy too! Why are you happy today?"
`
	replyText, err := s.llmProvider.Chat(ctx, systemPrompt, userText)
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp llm failed", "error", err)
		return nil, err
	}
	logger.InfoContext(ctx, "chat mvp llm reply", "replyText", replyText)
	// 步骤 4：TTS 合成
	ttsAudio, err := s.ttsProvider.Synthesize(ctx, replyText, nil)
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp tts failed", "error", err)
		return nil, err
	}
	logger.InfoContext(ctx, "chat mvp tts audio generated", "audioSize", len(ttsAudio))
	// 步骤 5：上传用户音频与 AI 音频到 OSS
	conversationID := req.ConversationID
	userMsgID := uuid.New()
	aiMsgID := uuid.New()
	userAudioKey := fmt.Sprintf("chat/%s/user_%s.%s", conversationID, userMsgID, audioType)
	aiAudioKey := fmt.Sprintf("chat/%s/ai_%s.mp3", conversationID, aiMsgID)

	var userAudioURL string
	var aiAudioURL string
	if s.ossProvider != nil {
		if url, uploadErr := s.ossProvider.UploadAudio(ctx, userAudioKey, req.AudioData); uploadErr != nil {
			logger.ErrorContext(ctx, "chat mvp upload user audio failed", "error", uploadErr)
		} else {
			userAudioURL = url
		}
		if url, uploadErr := s.ossProvider.UploadAudio(ctx, aiAudioKey, ttsAudio); uploadErr != nil {
			logger.ErrorContext(ctx, "chat mvp upload ai audio failed", "error", uploadErr)
		} else {
			aiAudioURL = url
		}
	} else {
		// 步骤 5：如果 OSS 未初始化，仅记录日志
		logger.ErrorContext(ctx, "chat mvp oss provider not initialized", "error", errors.New("oss provider nil"))
	}
	logger.InfoContext(ctx, "chat mvp oss audio urls", "userAudioURL", userAudioURL, "aiAudioURL", aiAudioURL)
	// 步骤 6：保存对话记录到数据库（失败不影响主流程）
	if s.conversationRepo != nil && s.messageRepo != nil {
		var userDuration *int
		var userAudioPtr *string
		if userAudioURL != "" {
			userAudioPtr = &userAudioURL
		}
		var aiAudioPtr *string
		if aiAudioURL != "" {
			aiAudioPtr = &aiAudioURL
		}

		var turnID = 1
		var nextSeq = 1
		// 步骤 6.1：获取当前最大 TurnID 并加 1 作为新的 TurnID
		if maxTurn, err := s.messageRepo.GetMaxTurnID(ctx, conversationID); err == nil {
			turnID = maxTurn + 1
		} else {
			logger.ErrorContext(ctx, "chat mvp get max turn id failed", "error", err)
		}
		// 步骤 6.2：获取当前最大 SequenceNumber 并加 1 作为新的 SequenceNumber
		if seq, err := s.messageRepo.GetNextSequenceNumber(ctx, conversationID); err == nil {
			nextSeq = seq
		} else {
			logger.ErrorContext(ctx, "chat mvp get next seq failed", "error", err)
		}

		messages := []*model.ConversationMessage{
			{
				ID:             userMsgID,
				ConversationID: conversationID,
				TurnID:         turnID,
				SenderType:     "user",
				MessageText:    userText,
				AudioURL:       userAudioPtr,
				AudioDuration:  userDuration,
				SequenceNumber: nextSeq,
			},
			{
				ID:             aiMsgID,
				ConversationID: conversationID,
				TurnID:         turnID,
				SenderType:     "ai",
				MessageText:    replyText,
				AudioURL:       aiAudioPtr,
				SequenceNumber: nextSeq + 1,
			},
		}
		if saveErr := s.messageRepo.BatchCreate(ctx, messages); saveErr != nil {
			logger.ErrorContext(ctx, "chat mvp save messages failed", "error", saveErr)
		}

		// 获取当前会话状态，更新消息数和时长
		var currentMsgCount = 0
		var currentDuration = 0
		if existingConv, err := s.conversationRepo.GetByID(ctx, conversationID); err == nil && existingConv != nil {
			currentMsgCount = existingConv.MessageCount
			currentDuration = existingConv.DurationSeconds
		}

		// 更新会话记录
		conversation := &model.VoiceConversation{
			ID:               conversationID,
			UserID:           req.UserID,
			Topic:            "General",
			DifficultyLevel:  difficultyLevel,
			ConversationType: conversationType,
			MessageCount:     currentMsgCount + 2,
			DurationSeconds:  currentDuration + len(req.AudioData)/16000,
			Status:           "active",
		}
		if saveErr := s.conversationRepo.Update(ctx, conversation); saveErr != nil {
			logger.ErrorContext(ctx, "chat mvp save conversation failed", "error", saveErr)
		}

	} else {
		logger.ErrorContext(ctx, "chat mvp repository not initialized", "error", errors.New("repository nil"))
	}
	logger.InfoContext(ctx, "chat mvp save conversation and messages success")
	// 步骤 7：返回音频
	return ttsAudio, nil
}

func (s *chatServiceImpl) SubmitChat(ctx context.Context, req *SubmitChatRequest) (string, error) {
	// 步骤 1：基础校验
	if req == nil {
		return "", errors.New("submit chat request is nil")
	}
	if len(req.AudioData) == 0 {
		return "", errors.New("audio data is empty")
	}
	if req.UserID == "" {
		return "", errors.New("user id is empty")
	}
	if req.ConversationID == "" {
		return "", errors.New("conversation_id is required (use /session/start to create)")
	}

	// 步骤 2：处理默认参数
	conversationType := strings.TrimSpace(req.ConversationType)
	if conversationType == "" {
		conversationType = "free_talk"
	}
	difficultyLevel := strings.TrimSpace(req.DifficultyLevel)
	if difficultyLevel == "" {
		difficultyLevel = "beginner"
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = "general"
	}
	audioType := strings.ToLower(strings.TrimSpace(req.AudioType))
	if audioType == "" {
		audioType = "wav"
	}

	// 步骤 3：速率限制检查
	if err := checkRateLimit(ctx, s.rateLimitFactory, "chat_submit", req.UserID, s.logger); err != nil {
		return "", err
	}

	// 步骤 4：验证会话存在
	if s.conversationRepo != nil {
		conv, err := s.conversationRepo.GetByID(ctx, req.ConversationID)
		if err != nil {
			logger.ErrorContext(ctx, "submit chat get session failed", "error", err)
			return "", fmt.Errorf("failed to verify session: %w", err)
		}
		if conv == nil {
			return "", fmt.Errorf("session not found: %s (use /session/start to create)", req.ConversationID)
		}
		if conv.UserID != req.UserID {
			return "", errors.New("session does not belong to current user")
		}
	}

	// 步骤 5：生成消息 ID 并预先落库
	userMsgID := uuid.New()
	aiMsgID := uuid.New()

	if s.messageRepo != nil {
		var turnID = 1
		var nextSeq = 1
		if maxTurn, err := s.messageRepo.GetMaxTurnID(ctx, req.ConversationID); err == nil {
			turnID = maxTurn + 1
		}
		if seq, err := s.messageRepo.GetNextSequenceNumber(ctx, req.ConversationID); err == nil {
			nextSeq = seq
		}

		messages := []*model.ConversationMessage{
			{
				ID:             userMsgID,
				ConversationID: req.ConversationID,
				TurnID:         turnID,
				SenderType:     "user",
				MessageText:    "", // 占位为空，等待异步结果覆盖
				SequenceNumber: nextSeq,
			},
			{
				ID:             aiMsgID,
				ConversationID: req.ConversationID,
				TurnID:         turnID,
				SenderType:     "ai",
				MessageText:    "", // 占位为空，等待异步结果覆盖
				SequenceNumber: nextSeq + 1,
			},
		}
		if saveErr := s.messageRepo.BatchCreate(ctx, messages); saveErr != nil {
			logger.ErrorContext(ctx, "submit chat pre-save placeholders failed", "error", saveErr)
			// 这里不强制中断，让后台继续处理
		}
	}

	// 步骤 6：序列化 Payload
	payloadStruct := chatPayloadData{
		AudioData:        req.AudioData,
		AudioType:        audioType,
		ConversationID:   req.ConversationID,
		ConversationType: conversationType,
		DifficultyLevel:  difficultyLevel,
		Topic:            topic,
		UserID:           req.UserID,
		UserMessageID:    userMsgID,
		AIMessageID:      aiMsgID,
	}
	payload, err := json.Marshal(&payloadStruct)
	if err != nil {
		return "", fmt.Errorf("marshal chat payload: %w", err)
	}

	// 步骤 7：构建 Task 并提交 (以 aiMsgID 作为 DomainID)
	task := &worker.Task{
		Type:     "chat",
		DomainID: aiMsgID,
		UserID:   req.UserID,
		Payload:  payload,
	}

	taskID, err := s.workerManager.SubmitTask(ctx, task)
	if err != nil {
		logger.ErrorContext(ctx, "submit chat task failed", "error", err)
		return "", fmt.Errorf("submit task: %w", err)
	}

	// 步骤 7：更新学习进度（占位）
	logger.InfoContext(ctx, "chat task submitted, daily progress updated",
		"task_id", taskID, "user_id", req.UserID, "conversation_id", req.ConversationID)

	return taskID, nil
}

// stageDescriptions 阶段中文描述
var stageDescriptions = map[string]string{
	"queued":    "任务已进入队列，等待处理",
	"asr":       "正在识别语音...",
	"llm":       "正在生成 AI 回复...",
	"tts":       "正在合成语音...",
	"oss":       "正在上传音频...",
	"db":        "正在保存记录...",
	"completed": "处理完成",
}

func (s *chatServiceImpl) GetChatResult(ctx context.Context, taskID string) (*ChatResultResponse, error) {
	if taskID == "" {
		return nil, errors.New("task_id is empty")
	}

	// 步骤 1：从缓存查询任务状态
	meta, err := s.taskCache.GetTaskMeta(ctx, taskID)
	if err != nil {
		logger.ErrorContext(ctx, "get task meta failed", "task_id", taskID, "error", err)
		return nil, fmt.Errorf("query task status: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 步骤 2：根据状态构建响应
	switch meta.Status {
	case "pending":
		stage := meta.CurrentStage
		if stage == "" {
			stage = "queued"
		}
		return &ChatResultResponse{
			TaskID:       taskID,
			Status:       "pending",
			CurrentStage: stage,
			Message:      stageDescriptions[stage],
		}, nil

	case "processing":
		stage := meta.CurrentStage
		msg := stageDescriptions[stage]
		if msg == "" {
			msg = "正在处理中..."
		}
		return &ChatResultResponse{
			TaskID:       taskID,
			Status:       "processing",
			CurrentStage: stage,
			Message:      msg,
		}, nil

	case "success":
		// 从 chatCache 获取完整结果
		resultKey := meta.ResultKey
		// resultKey 格式: chat:result:{aiMsgID}，提取 aiMsgID 部分
		var aiMsgID string
		fmt.Sscanf(resultKey, "chat:result:%s", &aiMsgID)

		chatResult, found, cacheErr := s.chatCache.GetChatResult(ctx, aiMsgID)
		if cacheErr != nil {
			logger.ErrorContext(ctx, "get chat result from cache failed", "task_id", taskID, "error", cacheErr)
			return nil, fmt.Errorf("get chat result: %w", cacheErr)
		}
		if !found {
			return nil, fmt.Errorf("chat result not found in cache for task: %s", taskID)
		}

		return &ChatResultResponse{
			TaskID:      taskID,
			Status:      "success",
			SessionID:   chatResult.ConversationID,
			UserInput:   &AudioInput{Text: chatResult.UserText, DurationMs: chatResult.DurationMs},
			AIResponse:  &AudioReply{Text: chatResult.AIReply, AudioURL: chatResult.AudioURL, DurationMs: chatResult.AIDurationMs},
			CreatedAt:   chatResult.CreatedAt,
			FeedbackURL: "/api/v1/chat/feedback",
		}, nil

	case "failed":
		return &ChatResultResponse{
			TaskID:       taskID,
			Status:       "failed",
			ErrorStage:   meta.CurrentStage,
			ErrorMessage: meta.Error,
			Message:      "任务处理失败，请重试或联系支持",
		}, nil

	default:
		logger.ErrorContext(ctx, "unknown task status", "task_id", taskID, "status", meta.Status)
		return nil, fmt.Errorf("unknown task status: %s", meta.Status)
	}
}

func (s *chatServiceImpl) GetChatHistory(ctx context.Context, req *ChatHistoryRequest) ([]*ConversationTurn, int64, error) {
	// TODO: Step2 实现
	// 1. 验证用户对该会话的访问权限
	// 2. 查询 conversation_messages 表
	// 3. 按 order 排序，分页返回
	return nil, 0, nil
}

func (s *chatServiceImpl) DeleteSession(ctx context.Context, sessionID, userID string) (int64, error) {
	// TODO: Step2 实现
	// 1. 验证用户对该会话的所有权
	// 2. 删除会话下所有消息
	// 3. 删除会话记录
	// 4. 返回删除的消息数量
	return 0, nil
}

func (s *chatServiceImpl) GetSessions(ctx context.Context, userID string, page, pageSize int) ([]*SessionSummary, int64, error) {
	// TODO: Step2 实现
	// 1. 查询 voice_conversations 表
	// 2. 按最后交互时间降序排列
	// 3. 分页返回会话摘要
	return nil, 0, nil
}

func (s *chatServiceImpl) SubmitChatFeedback(ctx context.Context, req *SubmitFeedbackRequest) error {
	// TODO: Step2 实现
	// 1. 验证 task_id / session_id 存在
	// 2. 保存反馈到数据库
	return nil
}

// ValidateFreetalk 验证 free talk 模式语音对话请求
func (s *chatServiceImpl) ValidateFreetalk(ctx context.Context, req *ValidateFreetalkRequest) error {
	// 步骤 1：速率限制检查
	if err := checkRateLimit(ctx, s.rateLimitFactory, "chat_submit", req.UserID, s.logger); err != nil {
		return err
	}

	// 步骤 2：验证会话存在
	if s.conversationRepo != nil {
		conv, err := s.conversationRepo.GetByID(ctx, req.ConversationID)
		if err != nil {
			logger.ErrorContext(ctx, "submit chat get session failed", "error", err)
			return fmt.Errorf("failed to verify session: %w", err)
		}
		if conv == nil {
			return fmt.Errorf("session not found: %s (use /session/start to create)", req.ConversationID)
		}
		if conv.UserID != req.UserID {
			return errors.New("session does not belong to current user")
		}
	}
	return nil
}

// HandleFreetalk 处理 free talk 模式语音对话请求
func (s *chatServiceImpl) HandleFreetalk(ctx context.Context, req *HandleFreetalkRequest) error {

	// 2. 创建并启动 Session
	// vadClient 由调用方（Handler 层）通过 chatServiceImpl.vadClient 注入，
	// 此处透传给 Session。
	session := NewSession(
		req.Conn,
		s.asrProvider,
		s.llmProvider,
		s.ttsProvider,
		req.ConversationID,
		req.UserID,
		s.vadClient,
		s.pronunciationService,
		s.reportService,
	)
	go session.Run()
	return nil
}

// SummaryContent LLM 生成的结构化摘要，字段全部为中文
type SummaryContent struct {
	Topic         string   `json:"topic"`         // 本次对话主要话题，1句话
	Highlights    []string `json:"highlights"`    // 孩子表现亮点，可为空数组
	Suggestions   []string `json:"suggestions"`   // 建议练习的地方，可为空数组
	Encouragement string   `json:"encouragement"` // 给孩子的鼓励评价，1-2句
}

// SessionSummaryResponse 摘要接口的完整响应体
type SessionSummaryResponse struct {
	SessionID string                  `json:"session_id"`
	UserID    string                  `json:"user_id"`
	StartedAt time.Time               `json:"started_at"`
	EndedAt   *time.Time              `json:"ended_at"`
	TurnCount int                     `json:"turn_count"`
	Messages  []model.FreeTalkMessage `json:"messages"` // 完整对话记录，按 seq 升序
	Summary   SummaryContent          `json:"summary"`
}

func (s *chatServiceImpl) GetFreetalkSummary(ctx context.Context, sessionID, userID string) (*SessionSummaryResponse, error) {

	dsn := "tohka:Tohka10xiang@tcp(8.155.145.36:3306)/oktalk?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("start session failed to connect database", "error", err)
	}
	// 校验 session 存在
	// 2. 第一次查询：校验 Session 存在且属于该用户
	var session model.FreeTalkSession
	err = db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session not found or access denied")
		}
		return nil, fmt.Errorf("query session error: %v", err)
	}
	// 读取完整消息记录
	var messages []model.FreeTalkMessage
	err = db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("seq asc"). // 必须按序号升序，确保摘要逻辑正确
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("query messages error: %v", err)
	}
	// ── 3. 调 LLM 生成结构化摘要 ────────────────────────────────
	summaryContent, err := s.generateSummary(ctx, messages)
	if err != nil {
		// 摘要生成失败时降级：只返回对话记录，摘要字段给默认值
		slog.Warn("[SummaryService] LLM summary failed, returning degraded response",
			"error", err,
			"session_id", sessionID,
		)
		summaryContent = SummaryContent{
			Topic:         "本次对话",
			Highlights:    []string{},
			Suggestions:   []string{},
			Encouragement: "你很棒，继续加油！",
		}
	}
	return &SessionSummaryResponse{
		SessionID: sessionID,
		UserID:    userID,
		StartedAt: session.StartedAt,
		EndedAt:   session.EndedAt,
		TurnCount: session.TurnCount,
		Messages:  messages,
		Summary:   summaryContent,
	}, nil
}

func (s *chatServiceImpl) generateSummary(ctx context.Context, messages []model.FreeTalkMessage) (SummaryContent, error) {
	if len(messages) == 0 {
		return SummaryContent{
			Topic:         "对话内容较少",
			Highlights:    []string{},
			Suggestions:   []string{},
			Encouragement: "期待下次更长的对话！",
		}, nil
	}
	userPrompt := buildSummaryUserPrompt(messages)

	raw, err := s.llmProvider.Chat(ctx, summarySystemPrompt(), userPrompt)
	if err != nil {
		return SummaryContent{}, fmt.Errorf("llm chat failed: %w", err)
	}

	return parseSummaryJSON(raw)
}

// summarySystemPrompt 摘要生成专用的 System Prompt，独立于对话用的 System Prompt
func summarySystemPrompt() string {
	return `You are an English learning coach assistant.
Your task is to analyze a conversation between an AI English coach named Mia and a Chinese child, then generate a session summary for parents.
 
IMPORTANT: You must respond with ONLY a valid JSON object. No markdown fences, no explanation, no extra text before or after.
 
The JSON must follow this exact structure:
{
  "topic": "本次对话的主要话题（1句话）",
  "highlights": ["亮点1", "亮点2"],
  "suggestions": ["建议1"],
  "encouragement": "给孩子的鼓励评价（1-2句，温暖友好）"
}
 
Rules:
- All field values must be in Chinese
- topic: one sentence describing the main subject discussed
- highlights: specific things the child did well — vocabulary used, sentences formed, topics engaged with. Empty array [] if nothing notable
- suggestions: specific areas to practice next time. Empty array [] if no clear suggestions  
- encouragement: always positive and warm, suitable for a child aged 6-12
- If the conversation is very short (less than 3 child utterances), still provide warm encouraging feedback`
}

// buildSummaryUserPrompt 将消息列表格式化为 LLM 可读的对话文本
func buildSummaryUserPrompt(messages []model.FreeTalkMessage) string {
	var sb strings.Builder
	sb.WriteString("Here is the conversation to analyze:\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case model.MessageRoleUser:
			sb.WriteString(fmt.Sprintf("Child: %s\n", msg.Content))
		case model.MessageRoleAssistant:
			sb.WriteString(fmt.Sprintf("Mia (AI Coach): %s\n", msg.Content))
		}
	}

	sb.WriteString("\nPlease generate the session summary JSON now.")
	return sb.String()
}

// parseSummaryJSON 解析 LLM 返回的 JSON 字符串，容错处理 markdown 包裹
func parseSummaryJSON(raw string) (SummaryContent, error) {
	raw = strings.TrimSpace(raw)
	// 去掉 LLM 可能添加的 markdown 代码块包裹
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var content SummaryContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return SummaryContent{}, fmt.Errorf("unmarshal summary JSON failed: %w (raw preview: %s)", err, previewStr(raw, 200))
	}

	// 保证数组字段非 nil，前端 JSON 序列化时 [] 而非 null
	if content.Highlights == nil {
		content.Highlights = []string{}
	}
	if content.Suggestions == nil {
		content.Suggestions = []string{}
	}

	return content, nil
}

// previewStr 截取字符串前 n 个字符用于日志输出
func previewStr(s string, n int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// ===================== ChatTaskProcessor =====================

// chatPayloadData 异步任务 payload 反序列化目标
type chatPayloadData struct {
	AudioData        []byte `json:"audio_data"`
	AudioType        string `json:"audio_type"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
	DifficultyLevel  string `json:"difficulty_level"`
	Topic            string `json:"topic"`
	UserID           string `json:"user_id"`
	UserMessageID    string `json:"user_message_id"`
	AIMessageID      string `json:"ai_message_id"`
}

// ChatTaskProcessor 实现 worker.TaskProcessor 接口
type ChatTaskProcessor struct {
	asrProvider      domain.ASRProvider
	llmProvider      domain.LLMProvider
	ttsProvider      domain.TTSProvider
	ossProvider      domain.OSSProvider
	conversationRepo db.VoiceConversationRepository
	messageRepo      db.ConversationMessageRepository
	taskCache        *cache.TaskCache
	logger           *slog.Logger
}

// NewChatTaskProcessor 创建 ChatTaskProcessor
func NewChatTaskProcessor(
	asr domain.ASRProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	repos *db.Repositories,
	taskCache *cache.TaskCache,
	logger *slog.Logger,
) *ChatTaskProcessor {
	var conversationRepo db.VoiceConversationRepository
	var messageRepo db.ConversationMessageRepository
	if repos != nil {
		conversationRepo = repos.VoiceConversation
		messageRepo = repos.ConversationMessage
	}
	return &ChatTaskProcessor{
		asrProvider:      asr,
		llmProvider:      llm,
		ttsProvider:      tts,
		ossProvider:      oss,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		taskCache:        taskCache,
		logger:           logger,
	}
}

// Process 实现 worker.TaskProcessor 接口
// 返回: (*cache.ChatResult, resultKey, error)
func (p *ChatTaskProcessor) Process(ctx context.Context, task *worker.Task) (interface{}, string, error) {
	// 反序列化payload
	var payload chatPayloadData
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return nil, "", fmt.Errorf("unmarshal payload: %w", err)
	}

	conversationID := payload.ConversationID
	audioType := payload.AudioType
	resultKey := fmt.Sprintf(cache.KeyChatResult, task.DomainID)

	// ===== A1: ASR 语音识别 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "asr")

	var asrResult string
	var asrErr error
	for retry := 0; retry < 3; retry++ {
		asrResult, asrErr = p.asrProvider.RecognizeAudio(ctx, payload.AudioData)
		if asrErr == nil {
			break
		}
		p.logger.Warn("ASR retry", slog.Int("attempt", retry+1), slog.String("error", asrErr.Error()))
		time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
	}
	if asrErr != nil {
		return nil, resultKey, fmt.Errorf("[asr] %w", asrErr)
	}
	userText := strings.TrimSpace(asrResult)
	if userText == "" {
		return nil, resultKey, errors.New("[asr] recognition result is empty")
	}

	// ===== A2: LLM 对话生成 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "llm")

	// 获取对话历史 (fallback to empty)
	var historyContext string
	if p.messageRepo != nil {
		msgs, err := p.messageRepo.GetByConversationID(ctx, conversationID)
		if err == nil && len(msgs) > 0 {
			// 取最后10条消息构建上下文
			start := 0
			if len(msgs) > 10 {
				start = len(msgs) - 10
			}
			var sb strings.Builder
			for _, m := range msgs[start:] {
				if m.SenderType == "user" {
					sb.WriteString("User: " + m.MessageText + "\n")
				} else {
					sb.WriteString("AI: " + m.MessageText + "\n")
				}
			}
			historyContext = sb.String()
		}
	}

	systemPrompt := fmt.Sprintf(`You are a friendly English teacher for Chinese kids (6-12 years old) learning English.
Topic: %s | Level: %s | Type: %s

CORE RULES:
1. Response length: Maximum 25 words (2 sentences)
2. Vocabulary: Use only simple, common words
3. Primary language: English
4. Always be encouraging and positive
5. If the child speaks Chinese, gently prompt in English

Conversation history:
%sCurrent user input: %s`, payload.Topic, payload.DifficultyLevel, payload.ConversationType, historyContext, userText)

	var replyText string
	var llmErr error
	replyText, llmErr = p.llmProvider.Chat(ctx, systemPrompt, userText)
	if llmErr != nil {
		p.logger.Warn("LLM failed, using fallback", slog.String("error", llmErr.Error()))
		replyText = "I didn't catch that. Could you say it again?"
	}
	if len(replyText) == 0 {
		replyText = "That's interesting! Tell me more."
	}

	// ===== A3: TTS 语音合成 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "tts")

	var ttsAudio []byte
	var ttsErr error
	for retry := 0; retry < 2; retry++ {
		ttsAudio, ttsErr = p.ttsProvider.Synthesize(ctx, replyText, nil)
		if ttsErr == nil {
			break
		}
		p.logger.Warn("TTS retry", slog.Int("attempt", retry+1), slog.String("error", ttsErr.Error()))
		time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
	}
	if ttsErr != nil {
		p.logger.Error("TTS failed, returning text-only result", slog.String("error", ttsErr.Error()))
		// TTS失败仅返回文本，不中断
	}

	// 步骤 5: 获取之前预留的记录并上传相关录音到 OSS
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "oss")

	userMsgID := payload.UserMessageID
	aiMsgID := payload.AIMessageID

	if userMsgID == "" || aiMsgID == "" {
		p.logger.Error("missing message id in payload, falling back to new uuid")
		userMsgID = uuid.New()
		aiMsgID = uuid.New()
	}

	userAudioKey := fmt.Sprintf("chat/%s/user_%s.%s", conversationID, userMsgID, audioType)
	aiAudioKey := fmt.Sprintf("chat/%s/ai_%s.mp3", conversationID, aiMsgID)

	var userAudioURL, aiAudioURL string
	if p.ossProvider != nil {
		if url, uploadErr := p.ossProvider.UploadAudio(ctx, userAudioKey, payload.AudioData); uploadErr != nil {
			p.logger.Error("upload user audio failed", slog.String("error", uploadErr.Error()))
		} else {
			userAudioURL = url
		}
		if len(ttsAudio) > 0 {
			if url, uploadErr := p.ossProvider.UploadAudio(ctx, aiAudioKey, ttsAudio); uploadErr != nil {
				p.logger.Error("upload ai audio failed", slog.String("error", uploadErr.Error()))
			} else {
				aiAudioURL = url
			}
		}
	}

	// ===== A6: 保存(更新)对话记录到数据库 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "db")

	if p.messageRepo != nil {
		var userAudioPtr, aiAudioPtr *string
		if userAudioURL != "" {
			userAudioPtr = &userAudioURL
		}
		if aiAudioURL != "" {
			aiAudioPtr = &aiAudioURL
		}
		var userDuration *int

		// Update User Message
		if userMsg, err := p.messageRepo.GetByID(ctx, userMsgID); err == nil && userMsg != nil {
			userMsg.MessageText = userText
			userMsg.AudioURL = userAudioPtr
			userMsg.AudioDuration = userDuration
			_ = p.messageRepo.Update(ctx, userMsg)
		} else {
			// Fallback: create if missing
			seq, _ := p.messageRepo.GetNextSequenceNumber(ctx, conversationID)
			turn, _ := p.messageRepo.GetMaxTurnID(ctx, conversationID)
			_ = p.messageRepo.Create(ctx, &model.ConversationMessage{
				ID:             userMsgID,
				ConversationID: conversationID,
				TurnID:         turn + 1,
				SenderType:     "user",
				MessageText:    userText,
				AudioURL:       userAudioPtr,
				AudioDuration:  userDuration,
				SequenceNumber: seq,
			})
		}

		// Update AI Message
		if aiMsg, err := p.messageRepo.GetByID(ctx, aiMsgID); err == nil && aiMsg != nil {
			aiMsg.MessageText = replyText
			aiMsg.AudioURL = aiAudioPtr
			_ = p.messageRepo.Update(ctx, aiMsg)
		} else {
			// Fallback: create if missing
			seq, _ := p.messageRepo.GetNextSequenceNumber(ctx, conversationID)
			turn, _ := p.messageRepo.GetMaxTurnID(ctx, conversationID)
			_ = p.messageRepo.Create(ctx, &model.ConversationMessage{
				ID:             aiMsgID,
				ConversationID: conversationID,
				TurnID:         turn,
				SenderType:     "ai",
				MessageText:    replyText,
				AudioURL:       aiAudioPtr,
				SequenceNumber: seq,
			})
		}

		// 更新会话的统计时长和消息数
		if p.conversationRepo != nil {
			if conv, err := p.conversationRepo.GetByID(ctx, conversationID); err == nil && conv != nil {
				// 获取实际消息数（考虑到可能是 fallback 创建的）
				msgCount, _ := p.messageRepo.CountByConversationID(ctx, conversationID)
				conv.MessageCount = int(msgCount)
				conv.DurationSeconds += len(payload.AudioData) / 16000
				_ = p.conversationRepo.Update(ctx, conv)
			}
		}
	}

	// ===== A7: 更新学习进度（占位） =====
	p.logger.Info("chat completion progress updated (placeholder)",
		slog.String("task_id", task.ID),
		slog.String("user_id", task.UserID),
	)

	// ===== A8: 构建结果 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "completed")

	chatResult := &cache.ChatResult{
		TaskID:         task.ID,
		ConversationID: conversationID,
		UserID:         task.UserID,
		UserText:       userText,
		UserAudioURL:   userAudioURL,
		DurationMs:     len(payload.AudioData) / 16000 * 1000,
		AIReply:        replyText,
		AudioURL:       aiAudioURL,
		CreatedAt:      time.Now().Unix(),
	}

	return chatResult, resultKey, nil
}

// ===================== ChatResultPersister =====================

// ChatResultPersister 实现 worker.ResultPersister 接口
type ChatResultPersister struct {
	logger *slog.Logger
}

// NewChatResultPersister 创建 ChatResultPersister
func NewChatResultPersister(logger *slog.Logger) *ChatResultPersister {
	return &ChatResultPersister{logger: logger}
}

// SaveResult 将结果持久化到数据库
// ChatTaskProcessor.Process 中已经完成了 DB 写入，这里只做日志记录
func (p *ChatResultPersister) SaveResult(ctx context.Context, task *worker.Task, result interface{}) error {
	p.logger.Info("chat result persisted (already saved in processor)",
		slog.String("task_id", task.ID),
		slog.String("type", task.Type),
	)
	return nil
}
