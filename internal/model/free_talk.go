package model

import "time"

// FreeTalkSession 对应 free_talk_sessions 表
type FreeTalkSession struct {
	ID        int64      `db:"id"         json:"id"`
	SessionID string     `db:"session_id" json:"session_id"`
	UserID    string     `db:"user_id"    json:"user_id"`
	StartedAt time.Time  `db:"started_at" json:"started_at"`
	EndedAt   *time.Time `db:"ended_at"   json:"ended_at"` // nil 表示异常断开未更新
	TurnCount int        `db:"turn_count" json:"turn_count"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// FreeTalkMessage 对应 free_talk_messages 表
type FreeTalkMessage struct {
	ID        int64     `db:"id"         json:"id"`
	SessionID string    `db:"session_id" json:"session_id"`
	Seq       int       `db:"seq"        json:"seq"`
	Role      string    `db:"role"       json:"role"` // MessageRoleUser | MessageRoleAssistant
	Content   string    `db:"content"    json:"content"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)
