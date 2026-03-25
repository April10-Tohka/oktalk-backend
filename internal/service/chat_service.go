// Package service 提供 AI 语音对话业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	domainLimiter "pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/worker"

	"github.com/gorilla/websocket"
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
	taskCache        *cache.TaskCache
	chatCache        *cache.ChatCache
	workerManager    *worker.Manager
	rateLimitFactory domainLimiter.SceneLimiterFactory
	logger           *slog.Logger
}

// NewChatService 创建 ChatService
func NewChatService(
	repos *db.Repositories,
	asr domain.ASRProvider,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	taskCache *cache.TaskCache,
	chatCache *cache.ChatCache,
	workerMgr *worker.Manager,
	rlFactory domainLimiter.SceneLimiterFactory,
	logger *slog.Logger,
) ChatService {
	var conversationRepo db.VoiceConversationRepository
	var messageRepo db.ConversationMessageRepository
	if repos != nil {
		conversationRepo = repos.VoiceConversation
		messageRepo = repos.ConversationMessage
	}
	return &chatServiceImpl{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		asrProvider:      asr,
		llmProvider:      llm,
		ttsProvider:      tts,
		ossProvider:      oss,
		taskCache:        taskCache,
		chatCache:        chatCache,
		workerManager:    workerMgr,
		rateLimitFactory: rlFactory,
		logger:           logger,
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

	conversationType := strings.TrimSpace(req.ConversationType)
	if conversationType == "" {
		conversationType = "free_talk"
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = "General"
	}

	conversationID := uuid.New()
	conversation := &model.VoiceConversation{
		ID:               conversationID,
		UserID:           req.UserID,
		Topic:            topic,
		ConversationType: conversationType,
		MessageCount:     0,
		DurationSeconds:  0,
		Status:           "active",
	}
	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		logger.ErrorContext(ctx, "start session create conversation failed", "error", err)
		return nil, err
	}

	return &StartSessionResponse{
		ConversationID:   conversationID,
		ConversationType: conversationType,
		Topic:            topic,
		Status:           "active",
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
	// 1. 更新会话状态为 active
	_ = s.conversationRepo.UpdateStatus(ctx, req.ConversationID, "active")

	// 2. 创建并启动 Session
	session := NewSession(
		req.Conn,
		s.asrProvider,
		s.llmProvider,
		s.ttsProvider,
		s.conversationRepo,
		s.messageRepo,
		req.ConversationID,
		req.UserID,
	)
	go session.Run()
	return nil
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

// ===================== 状态机 =====================

type sessionState int

const (
	stateIdle       sessionState = iota // 接收并转发用户音频
	stateAISpeaking                     // AI 回复中，丢弃用户音频
)

// ===================== Text Frame 类型常量 =====================

const (
	// MsgTypeLLMToken LLM 流式 token（后端 → App）
	// 每个 token 作为一条 text frame 推送，App 可实时展示打字效果
	MsgTypeLLMToken = "llm_token"

	// MsgTypeTurnEnd 本轮 AI 回复结束（后端 → App）
	// App 收到后可恢复录音
	MsgTypeTurnEnd = "turn_end"

	// MsgTypeError 错误通知（后端 → App）
	// 包含 code 和 message 字段
	MsgTypeError = "error"

	// MsgTypeASRText ASR 最终识别文本（后端 → App）
	// 用于在 App 端展示用户说的话
	MsgTypeASRText = "asr_text"

	// MsgTypeASRPartial ASR 中间识别文本（后端 → App）
	// 用于实时展示识别过程
	MsgTypeASRPartial = "asr_partial"
)

// ===================== App → 后端（Text Frame）=====================

// IncomingMessage App 发来的 Text Frame 结构
type IncomingMessage struct {
	// Type 消息类型: "start" / "stop"
	// "start": 开始/恢复 Free Talk 会话
	// "stop": 结束 Free Talk 会话
	Type string `json:"type"`

	// ConversationID 会话 ID（start 时必传）
	ConversationID string `json:"conversation_id,omitempty"`
}

// ===================== 后端 → App（Text Frame）=====================

// OutgoingMessage 后端推给 App 的 Text Frame 结构
type OutgoingMessage struct {
	// Type 消息类型（见上方常量）
	Type string `json:"type"`

	// Text 文本内容
	// - llm_token: LLM 生成的增量 token
	// - asr_text: ASR 最终识别文本
	// - asr_partial: ASR 中间识别文本
	// - turn_end: 空
	Text string `json:"text,omitempty"`

	// Code 错误码（仅 error 时使用）
	Code string `json:"code,omitempty"`

	// Message 错误信息（仅 error 时使用）
	Message string `json:"message,omitempty"`
}

// ===================== Binary Frame =====================
// Binary Frame 携带 PCM 裸音频数据，无任何包装：
// - App → 后端：用户录音 PCM，16kHz 单声道 16bit
// - 后端 → App：TTS 合成音频（格式由配置决定，默认 PCM）

// ===================== WebSocket 内部消息类型 =====================

// wsMessage 内部 WebSocket 消息封装
// 用于 writerGoroutine 串行化所有写操作
type wsMessage struct {
	// messageType websocket.TextMessage 或 websocket.BinaryMessage
	messageType int

	// data 消息内容（JSON 或二进制音频）
	data []byte
}

// ===================== Session 结构体 =====================

// Session 管理单个 Free Talk WebSocket 会话的完整生命周期
type Session struct {
	appConn *websocket.Conn

	// 注入的 domain 接口（由 Handler 创建后传入）
	asrProvider domain.ASRProvider
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider

	// 状态机
	state   sessionState
	stateMu sync.Mutex

	// 会话信息
	conversationID   string
	userID           string
	conversationRepo db.VoiceConversationRepository
	messageRepo      db.ConversationMessageRepository

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSession(
	appConn *websocket.Conn,
	asrProvider domain.ASRProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	conversationRepo db.VoiceConversationRepository,
	messageRepo db.ConversationMessageRepository,
	conversationID string,
	userID string,
) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		appConn:          appConn,
		asrProvider:      asrProvider,
		llmProvider:      llmProvider,
		ttsProvider:      ttsProvider,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		conversationID:   conversationID,
		userID:           userID,
		ctx:              ctx,
		cancel:           cancel,
		state:            stateIdle,
	}
}

func (s *Session) Run() error {
	audioChan := make(chan []byte, 1024)              // App音频 → ASR goroutine（转发PCM给ASR）
	llmInputChan := make(chan string, 1024)           // ASR goroutine → LLM goroutine（触发LLM，携带识别文本）
	llmOutputChan := make(chan domain.LLMChunk, 1024) // LLM goroutine → TTS goroutine（流式token投喂TTS）
	writeChan := make(chan wsMessage, 1024)           // 所有goroutine → Writer goroutine（统一写App WebSocket）
	ttsNewTurnChan := make(chan struct{}, 1024)       // ASR goroutine → TTS goroutine（触发新一轮TTS任务）

	go s.readerGoroutine(audioChan)
	go s.writerGoroutine(writeChan)
	go s.asrGoroutine(audioChan, llmInputChan, ttsNewTurnChan)
	go s.llmGoroutine(llmInputChan, llmOutputChan, writeChan)
	go s.ttsGoroutine(llmOutputChan, ttsNewTurnChan, writeChan)
	return nil
}

// ===================== ① writerGoroutine =====================

// writerGoroutine 唯一负责写 appConn 的 goroutine
// 从 writeChan 取消息 → appConn.WriteMessage
// writeChan 关闭或 ctx 取消时退出
func (s *Session) writerGoroutine(writeChan <-chan wsMessage) {
	for msg := range writeChan {
		if err := s.appConn.WriteMessage(msg.messageType, msg.data); err != nil {
			slog.Error("[FreeTalk] Write to app failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
			s.cancel()
			return
		}
	}
}

// ===================== ② appReaderGoroutine =====================

// appReaderGoroutine 持续读取 App 发来的帧
// Text Frame → 解析 IncomingMessage（start/stop 控制指令）
// Binary Frame（PCM）→ 根据状态转发到 ASR 或丢弃
func (s *Session) readerGoroutine(audioChan chan<- []byte) {
	defer s.cancel()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		slog.Info("准备readmessage")
		messageType, data, err := s.appConn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("[FreeTalk] App disconnected normally",
					"conversation_id", s.conversationID,
				)
				return
			}
			slog.Error("[FreeTalk] Read from app failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			s.handleTextFrame(data)

		case websocket.BinaryMessage:
			audioChan <- data

		default:
			slog.Warn("[FreeTalk] Unexpected message type",
				"type", messageType,
				"conversation_id", s.conversationID,
			)
		}
	}
}

// ===================== ③ asrGoroutine =====================
func (s *Session) asrGoroutine(audioChan <-chan []byte, llmInputChan chan<- string, ttsNewTurnChan chan<- struct{}) {
	s.asrProvider.ConnectASR(s.ctx, audioChan, llmInputChan, ttsNewTurnChan)
}

func (s *Session) llmGoroutine(llmInputChan <-chan string, llmOutputChan chan<- domain.LLMChunk, writeChan chan<- wsMessage) {
	// 创建新对话
	convID, err := s.llmProvider.NewConversation(s.ctx)
	if err != nil {
		slog.Error("[FreeTalk] New conversation failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		return
	}
	// 启动一个goroutine，持续获取llmIntputChan中的数据，调用llmProvider.ChatStream，将结果写入llmOutputChan
	go func() {
		for input := range llmInputChan {
			stream := s.llmProvider.ChatStream(s.ctx, convID, input)

			// 处理流式响应
			for stream.Next() {
				event := stream.Current()
				// 日志输出
				slog.Info("[FreeTalk] LLM stream event",
					"event.Delta", event.Delta,
					"event", event,
				)
				llmOutputChan <- domain.LLMChunk{
					Text:   event.Delta,
					IsDone: false,
				}
				writeChan <- wsMessage{
					messageType: websocket.TextMessage,
					data:        []byte(event.Delta),
				}
			}
			if stream.Err() != nil {
				slog.Error("[FreeTalk] Chat stream failed",
					"error", stream.Err(),
				)
				return
			}

		}
	}()

}

func (s *Session) ttsGoroutine(llmOutputChan <-chan domain.LLMChunk, ttsNewTurnChan <-chan struct{}, writeChan chan<- wsMessage) {
	//  建立websocket连接。需要使用ttsProvider.ConnectTTS()方法。
	ttsConn, err := s.ttsProvider.ConnectTTS(s.ctx)
	if err != nil {
		slog.Error("[FreeTalk] Connect TTS failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		return
	}

	// 启动一个goroutine，持续获取ttsNewTurnChan中的数据。
	// 每当收到一个新事件，代表这是新的一轮tts合成音频的任务。需要发送run-task指令
	go func() {
		for range ttsNewTurnChan {
			// 生成任务ID
			taskID := uuid.New()

			// 发送run-task指令
			runTaskCmd := map[string]interface{}{
				"header": map[string]interface{}{
					"action":    "run-task",
					"task_id":   taskID,
					"streaming": "duplex",
				},
				"payload": map[string]interface{}{
					"task_group": "audio",
					"task":       "tts",
					"function":   "SpeechSynthesizer",
					"model":      "cosyvoice-v3-flash",
					"parameters": map[string]interface{}{
						"text_type":   "PlainText",
						"voice":       "longanyang",
						"format":      "mp3",
						"sample_rate": 22050,
						"volume":      50,
						"rate":        1,
						"pitch":       1,
						// 如果enable_ssml设为true，只允许发送一次continue-task指令，否则会报错“Text request limit violated, expected 1.”
						"enable_ssml": false,
					},
					"input": map[string]interface{}{},
				},
			}

			runTaskJSON, _ := json.Marshal(runTaskCmd)
			ttsConn.WriteMessage(websocket.TextMessage, runTaskJSON)
			if err != nil {
				slog.Error("[FreeTalk] Send run task failed",
					"error", err,
				)
				return
			}
			// 创建一个通道，用于接收task-started事件
			taskStartedChan := make(chan struct{}, 1)
			// 启动一个goroutine异步接收WebSocket消息
			go func() {
				for {
					messageType, message, err := ttsConn.ReadMessage()
					if err != nil {
						slog.Error("[FreeTalk] Read tts message failed",
							"error", err,
						)
						return
					}
					if messageType == websocket.BinaryMessage {
						writeChan <- wsMessage{
							messageType: websocket.BinaryMessage,
							data:        message,
						}
						continue
					} else if messageType == websocket.TextMessage {
						var event wsEvent
						if err := json.Unmarshal(message, &event); err != nil {
							slog.Error("[FreeTalk] Parse event failed",
								"error", err,
							)
						}
						if event.Header.Event == "task-started" {
							taskStartedChan <- struct{}{}
						} else if event.Header.Event == "task-failed" {
							slog.Error("[FreeTalk] Task failed",
								"error", event.Header.ErrorMessage,
							)
							return
						} else if event.Header.Event == "task-finished" {
							slog.Info("[FreeTalk] Task finished",
								"task_id", event.Header.TaskID,
							)
							return
						}
					}
				}
			}()
			// 启动一个goroutine，持续获取llmOutputChan中的数据。
			go func() {
				<-taskStartedChan
				for chunk := range llmOutputChan {
					// 发送continue-task指令
					continueTaskCmd := map[string]interface{}{
						"header": map[string]interface{}{
							"action":    "continue-task",
							"task_id":   taskID,
							"streaming": "duplex",
						},
						"payload": map[string]interface{}{
							"input": map[string]interface{}{
								"text": chunk.Text,
							},
						},
					}

					continueTaskJSON, _ := json.Marshal(continueTaskCmd)
					err = ttsConn.WriteMessage(websocket.TextMessage, continueTaskJSON)
					if err != nil {
						slog.Error("[FreeTalk] Send continue task failed",
							"error", err,
						)
						return
					}

					if chunk.IsDone {
						// 发送finish-task指令
						finishTaskCmd := map[string]interface{}{
							"header": map[string]interface{}{
								"action":    "finish-task",
								"task_id":   taskID,
								"streaming": "duplex",
							},
							"payload": map[string]interface{}{
								"input": map[string]interface{}{},
							},
						}

						finishTaskJSON, _ := json.Marshal(finishTaskCmd)

						err = ttsConn.WriteMessage(websocket.TextMessage, finishTaskJSON)
						if err != nil {
							slog.Error("[FreeTalk] Send finish task failed",
								"error", err,
							)
							return
						}
						return
					}
				}

			}()
		}
	}()

}

// handleTextFrame 处理 App 发来的文本帧
func (s *Session) handleTextFrame(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("[FreeTalk] Parse text frame failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		return
	}

	switch msg.Type {
	case "stop":
		slog.Info("[FreeTalk] Received stop command",
			"conversation_id", s.conversationID,
		)
		s.cancel()

	default:
		slog.Warn("[FreeTalk] Unknown text frame type",
			"type", msg.Type,
			"conversation_id", s.conversationID,
		)
	}
}

// handleBinaryFrame 处理 App 发来的二进制帧（PCM 音频）
func (s *Session) handleBinaryFrame(data []byte) {

}

// wsEvent WebSocket 事件（请求和响应的统一结构）
type wsEvent struct {
	Header  wsHeader  `json:"header"`
	Payload wsPayload `json:"payload"`
}

// wsHeader 事件头部
type wsHeader struct {
	// Action 请求动作: "run-task", "continue-task", "finish-task"
	Action string `json:"action,omitempty"`

	// TaskID 任务唯一标识
	TaskID string `json:"task_id"`

	// Streaming 流式模式: "duplex"（双工）
	Streaming string `json:"streaming,omitempty"`

	// Event 响应事件类型: "task-started", "result-generated", "task-finished", "task-failed"
	Event string `json:"event,omitempty"`

	// ErrorCode 错误码（仅 task-failed 时有值）
	ErrorCode string `json:"error_code,omitempty"`

	// ErrorMessage 错误信息（仅 task-failed 时有值）
	ErrorMessage string `json:"error_message,omitempty"`

	// Attributes 附加属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// wsPayload 事件负载
type wsPayload struct {
	// 请求字段
	TaskGroup  string   `json:"task_group,omitempty"`
	Task       string   `json:"task,omitempty"`
	Function   string   `json:"function,omitempty"`
	Model      string   `json:"model,omitempty"`
	Parameters wsParams `json:"parameters,omitempty"`
	Input      wsInput  `json:"input"`

	// 响应字段
	Output wsOutput `json:"output,omitempty"`
	Usage  *wsUsage `json:"usage,omitempty"`
}

// wsParams TTS 合成参数
type wsParams struct {
	// TextType 文本类型: "PlainText"
	TextType string `json:"text_type,omitempty"`

	// Voice 音色: "longanyang", "longxiaochun" 等
	Voice string `json:"voice,omitempty"`

	// Format 音频格式: "mp3", "wav", "pcm"
	Format string `json:"format,omitempty"`

	// SampleRate 采样率: 8000, 16000, 22050, 24000, 48000
	SampleRate int `json:"sample_rate,omitempty"`

	// Volume 音量: 0-100
	Volume int `json:"volume,omitempty"`

	// Rate 语速: 0.5-2.0
	Rate float64 `json:"rate,omitempty"`

	// Pitch 音调: 0.5-2.0
	Pitch float64 `json:"pitch,omitempty"`

	// EnableSSML 是否启用 SSML（启用后只允许发送一次 continue-task）
	EnableSSML bool `json:"enable_ssml,omitempty"`
}

// wsInput 输入内容
type wsInput struct {
	// Text 待合成文本（用于 continue-task 指令）
	Text string `json:"text,omitempty"`
}

// wsOutput 输出内容（用于 result-generated 事件）
type wsOutput struct {
	// 部分 TTS 事件可能在此返回额外信息
}

// wsUsage 计费信息
type wsUsage struct {
	// Characters 已消耗字符数
	Characters int `json:"characters,omitempty"`

	// Duration 音频时长（秒）
	Duration int `json:"duration,omitempty"`
}
