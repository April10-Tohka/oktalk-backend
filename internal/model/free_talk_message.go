package model

import "time"

// FreeTalkMessage 对应 free_talk_messages 表
type FreeTalkMessage struct {
	ID string `gorm:"primaryKey;type:varchar(36);not null" json:"id"`
	// 为 SessionID 添加索引，方便按会话快速检索聊天记录
	SessionID string    `gorm:"type:varchar(36);index;not null"      json:"session_id"`
	Seq       int       `gorm:"type:int;not null"                    json:"seq"`
	Role      string    `gorm:"type:varchar(20);not null"             json:"role"` // user | assistant
	Content   string    `gorm:"type:text"                            json:"content"`
	CreatedAt time.Time `gorm:"autoCreateTime"                       json:"created_at"`
}

const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)

// TableName 表名
func (FreeTalkMessage) TableName() string { return "free_talk_messages" }
