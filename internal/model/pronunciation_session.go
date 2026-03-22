package model

import "time"

// PronunciationSession 发音练习 v2 会话
type PronunciationSession struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"                    json:"id"`
	UserID       string    `gorm:"type:varchar(36);not null;index"                 json:"user_id"`
	UnitID       string    `gorm:"type:varchar(50);not null"                      json:"unit_id"`
	CurrentIndex int       `gorm:"type:int;not null;default:1"                    json:"current_index"`
	Status       string    `gorm:"type:varchar(20);not null;default:'ongoing';index" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"                                 json:"updated_at"`
}

// TableName 表名
func (PronunciationSession) TableName() string { return "pronunciation_sessions" }
