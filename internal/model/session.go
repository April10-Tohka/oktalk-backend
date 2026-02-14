// Package model 定义会话模型（用户登录会话）
package model

import (
	"time"
)

// Session 用户登录会话模型
type Session struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	UserID    string    `json:"user_id" gorm:"index;size:36;not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;size:255;not null"`
	UserAgent string    `json:"user_agent" gorm:"size:255"`
	IP        string    `json:"ip" gorm:"size:50"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
