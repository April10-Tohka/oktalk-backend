// Package service 提供 AI 语音对话业务逻辑
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
)

// ===== 请求结构 =====

// ChatMVPRequest MVP 同步语音对话请求
type ChatMVPRequest struct {
	AudioData        []byte
	AudioType        string // wav / mp3
	ConversationType string // free_talk / question_answer
	DifficultyLevel  string // beginner / intermediate / advanced
	UserID           string
}

// SubmitChatRequest 异步语音对话提交请求
type SubmitChatRequest struct {
	AudioData    []byte
	AudioType    string
	SessionID    string
	UserLanguage string
	TopicID      string
	UserID       string
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

// ===== 响应结构 =====

// ChatResultResponse 语音对话处理结果
type ChatResultResponse struct {
	TaskID       string      `json:"task_id"`
	Status       string      `json:"status"`
	Progress     int         `json:"progress,omitempty"`
	CurrentStage string      `json:"current_stage,omitempty"`
	UserInput    *AudioInput `json:"user_input,omitempty"`
	AIResponse   *AudioReply `json:"ai_response,omitempty"`
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
}

// ===== 空实现 =====

// chatServiceImpl Chat Service 实现（暂为空实现）
type chatServiceImpl struct {
	// Step2 注入依赖
	conversationRepo db.VoiceConversationRepository
	messageRepo      db.ConversationMessageRepository
	asrProvider      domain.ASRProvider
	llmProvider      domain.LLMProvider
	ttsProvider      domain.TTSProvider
	ossProvider      domain.OSSProvider
	logger           *slog.Logger
}

// NewChatService 创建 ChatService
func NewChatService(repos *db.Repositories, asr domain.ASRProvider, llm domain.LLMProvider, tts domain.TTSProvider, oss domain.OSSProvider, logger *slog.Logger) ChatService {
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
		logger:           logger,
	}
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
	asrResult, err := s.asrProvider.RecognizeAudio(ctx, req.AudioData, audioType, 16000)
	if err != nil {
		logger.ErrorContext(ctx, "chat mvp asr failed", "error", err)
		return nil, err
	}
	userText := strings.TrimSpace(asrResult.Text)
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
	logger.InfoContext(ctx, "开始上传用户音频与 AI 音频到 OSS")
	// 步骤 5：上传用户音频与 AI 音频到 OSS
	conversationID := uuid.New()
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
	logger.InfoContext(ctx, "保存对话记录到数据库")
	// 步骤 6：保存对话记录到数据库（失败不影响主流程）
	if s.conversationRepo != nil && s.messageRepo != nil {
		conversation := &model.VoiceConversation{
			ID:               conversationID,
			UserID:           req.UserID,
			Topic:            "General",
			DifficultyLevel:  difficultyLevel,
			ConversationType: conversationType,
			MessageCount:     2,
			DurationSeconds:  asrResult.Duration,
			Status:           "completed",
		}
		if saveErr := s.conversationRepo.Create(ctx, conversation); saveErr != nil {
			logger.ErrorContext(ctx, "chat mvp save conversation failed", "error", saveErr)
		} else {
			var userDuration *int
			if asrResult.Duration > 0 {
				userDuration = &asrResult.Duration
			}
			var userAudioPtr *string
			if userAudioURL != "" {
				userAudioPtr = &userAudioURL
			}
			var aiAudioPtr *string
			if aiAudioURL != "" {
				aiAudioPtr = &aiAudioURL
			}

			messages := []*model.ConversationMessage{
				{
					ID:             userMsgID,
					ConversationID: conversationID,
					SenderType:     "user",
					MessageText:    userText,
					AudioURL:       userAudioPtr,
					AudioDuration:  userDuration,
					SequenceNumber: 1,
				},
				{
					ID:             aiMsgID,
					ConversationID: conversationID,
					SenderType:     "ai",
					MessageText:    replyText,
					AudioURL:       aiAudioPtr,
					SequenceNumber: 2,
				},
			}
			if saveErr := s.messageRepo.BatchCreate(ctx, messages); saveErr != nil {
				logger.ErrorContext(ctx, "chat mvp save messages failed", "error", saveErr)
			}
		}
	} else {
		logger.ErrorContext(ctx, "chat mvp repository not initialized", "error", errors.New("repository nil"))
	}
	logger.InfoContext(ctx, "chat mvp save conversation and messages success")
	// 步骤 7：返回音频
	return ttsAudio, nil
}

func (s *chatServiceImpl) SubmitChat(ctx context.Context, req *SubmitChatRequest) (string, error) {
	// TODO: Step3 实现异步任务
	// 1. 生成 task_id
	// 2. 创建异步任务（ASR → LLM → TTS）
	// 3. 将任务提交到队列
	// 4. 返回 task_id
	return "", nil
}

func (s *chatServiceImpl) GetChatResult(ctx context.Context, taskID string) (*ChatResultResponse, error) {
	// TODO: Step3 实现
	// 1. 从缓存/数据库查询任务状态
	// 2. 如果完成，返回完整结果
	// 3. 如果处理中，返回进度信息
	// 4. 如果失败，返回错误信息
	return nil, nil
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
