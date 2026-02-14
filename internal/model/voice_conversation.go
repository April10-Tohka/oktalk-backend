// Package model 定义语音对话相关数据模型
package model

import (
	"time"
)

// VoiceConversation 语音对话记录表
// 存储用户的 AI 语音对话会话
type VoiceConversation struct {
	// ID 对话 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// UserID 用户 ID，外键
	UserID string `gorm:"index;type:varchar(36);not null" json:"user_id" validate:"required,uuid"`
	// Topic 对话主题（如 "Hello"、"Daily Routine"）
	Topic string `gorm:"type:varchar(200);not null" json:"topic" validate:"required,max=200"`
	// DifficultyLevel 难度等级：beginner/intermediate/advanced
	DifficultyLevel string `gorm:"index;type:enum('beginner','intermediate','advanced');default:'beginner';not null" json:"difficulty_level" validate:"required,oneof=beginner intermediate advanced"`
	// ConversationType 对话类型：free_talk/scenario/question_answer
	ConversationType string `gorm:"index;type:enum('free_talk','scenario','question_answer');default:'free_talk';not null" json:"conversation_type" validate:"required,oneof=free_talk scenario question_answer"`
	// MessageCount 消息总数（包括用户和 AI）
	MessageCount int `gorm:"type:int;default:0;not null" json:"message_count" validate:"gte=0"`
	// DurationSeconds 对话时长（秒）
	DurationSeconds int `gorm:"type:int;default:0;not null" json:"duration_seconds" validate:"gte=0"`
	// Status 状态：active/completed/paused
	Status string `gorm:"index;type:enum('active','completed','paused');default:'active';not null" json:"status" validate:"required,oneof=active completed paused"`
	// LanguagePair 语言对：en-zh/en-es 等
	LanguagePair string `gorm:"type:varchar(20);default:'en-zh';not null" json:"language_pair" validate:"required,max=20"`
	// Summary 对话摘要
	Summary *string `gorm:"type:text" json:"summary,omitempty"`
	// AIModel AI 模型名称
	AIModel string `gorm:"type:varchar(100);default:'gpt-4';not null" json:"ai_model" validate:"required,max=100"`
	// Score 对话评分（0-100）
	Score *int `gorm:"type:int" json:"score,omitempty" validate:"omitempty,min=0,max=100"`
	// Feedback AI 反馈内容
	Feedback *string `gorm:"type:text" json:"feedback,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`
	// DeletedAt 软删除时间
	DeletedAt *time.Time `gorm:"type:timestamp" json:"deleted_at,omitempty"`

	// 关联
	User     *User                  `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Messages []*ConversationMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

// TableName 指定表名
func (VoiceConversation) TableName() string {
	return "voice_conversations"
}

// ConversationMessage 对话消息明细表
// 存储对话中的每一条消息（用户或 AI）
type ConversationMessage struct {
	// ID 消息 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// ConversationID 对话 ID，外键
	ConversationID string `gorm:"index;type:varchar(36);not null" json:"conversation_id" validate:"required,uuid"`
	// SenderType 发送者类型：user/ai
	SenderType string `gorm:"index;type:enum('user','ai');not null" json:"sender_type" validate:"required,oneof=user ai"`
	// SenderID 发送者 ID（仅当 sender_type=user 时有值）
	SenderID *string `gorm:"type:varchar(36)" json:"sender_id,omitempty" validate:"omitempty,uuid"`
	// MessageText 消息文本内容
	MessageText string `gorm:"type:longtext;not null" json:"message_text" validate:"required"`
	// MessageAudioURL 消息音频 URL（用户语音消息）
	MessageAudioURL *string `gorm:"type:varchar(500)" json:"message_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// MessageAudioDuration 音频时长（秒）
	MessageAudioDuration *int `gorm:"type:int" json:"message_audio_duration,omitempty" validate:"omitempty,gte=0"`
	// AIResponseText AI 回复文本（仅当 sender_type=ai 时有值）
	AIResponseText *string `gorm:"type:longtext" json:"ai_response_text,omitempty"`
	// AIResponseAudioURL AI 回复音频 URL
	AIResponseAudioURL *string `gorm:"type:varchar(500)" json:"ai_response_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// PronunciationScore 发音评分（仅对用户消息有值，0-100）
	PronunciationScore *int `gorm:"type:int" json:"pronunciation_score,omitempty" validate:"omitempty,min=0,max=100"`
	// PronunciationFeedback 发音反馈
	PronunciationFeedback *string `gorm:"type:text" json:"pronunciation_feedback,omitempty"`
	// SequenceNumber 消息序号（对话内的顺序）
	SequenceNumber int `gorm:"type:int;not null" json:"sequence_number" validate:"required,gte=0"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	Conversation *VoiceConversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
}

// TableName 指定表名
func (ConversationMessage) TableName() string {
	return "conversation_messages"
}
