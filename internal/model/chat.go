// Package model 定义对话相关数据模型
package model

import (
	"time"
)

// ChatSession 对话会话模型
type ChatSession struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	UserID    string    `json:"user_id" gorm:"index;size:36;not null"`
	Title     string    `json:"title" gorm:"size:100"`
	Scenario  string    `json:"scenario" gorm:"size:50"` // 对话场景
	Status    int       `json:"status" gorm:"default:1"` // 1:活跃 0:已结束
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage 对话消息模型
type ChatMessage struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	SessionID string    `json:"session_id" gorm:"index;size:36;not null"`
	Role      string    `json:"role" gorm:"size:20;not null"` // user, assistant, system
	Content   string    `json:"content" gorm:"type:text;not null"`
	AudioURL  string    `json:"audio_url" gorm:"size:255"` // 语音消息 URL
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// ChatSessionWithMessages 包含消息的会话
type ChatSessionWithMessages struct {
	ChatSession
	Messages []*ChatMessage `json:"messages"`
}
