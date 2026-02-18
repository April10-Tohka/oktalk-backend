// Package service 提供 AI 语音对话业务逻辑
package service

import (
	"context"
	"log/slog"
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
	// TODO: Step2 注入依赖
	// conversationRepo db.VoiceConversationRepository
	// messageRepo      db.ConversationMessageRepository
	// asrProvider       domain.ASRProvider
	// llmProvider       domain.LLMProvider
	// ttsProvider       domain.TTSProvider
	// ossProvider       domain.OSSProvider
	logger *slog.Logger
}

// NewChatService 创建 ChatService
func NewChatService(logger *slog.Logger) ChatService {
	return &chatServiceImpl{logger: logger}
}

func (s *chatServiceImpl) ChatMVP(ctx context.Context, req *ChatMVPRequest) ([]byte, error) {
	// TODO: Step2 实现
	// 1. ASR: 识别音频 → 文本
	// 2. LLM: 生成 AI 回复文本
	// 3. TTS: 合成 AI 回复 → 音频
	// 4. 保存对话记录到数据库
	// 5. 返回音频二进制数据
	return nil, nil
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
