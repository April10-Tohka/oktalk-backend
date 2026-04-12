package model

import "time"

// FreeTalkSession 对应 free_talk_sessions 表
type FreeTalkSession struct {
	// ID 作为物理主键
	ID string `gorm:"primaryKey;type:varchar(36);not null" json:"id"`
	// SessionID 作为业务唯一标识，添加唯一索引
	SessionID string `gorm:"type:varchar(36);uniqueIndex;not null" json:"session_id"`
	// 为 UserID 添加索引，方便查询某个用户的历史会话
	UserID    string     `gorm:"type:varchar(36);index;not null"      json:"user_id"`
	StartedAt time.Time  `gorm:"type:datetime;precision:3"            json:"started_at"`
	EndedAt   *time.Time `gorm:"type:datetime;precision:3"            json:"ended_at"`
	TurnCount int        `gorm:"type:int;default:0"                   json:"turn_count"`
	CreatedAt time.Time  `gorm:"autoCreateTime"                       json:"created_at"`
}

// TableName 表名
func (FreeTalkSession) TableName() string { return "free_talk_sessions" }
