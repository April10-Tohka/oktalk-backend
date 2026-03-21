package model

import "time"

// SceneMessage 场景引导单轮消息记录
type SceneMessage struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"      json:"id"`
	SessionID    string    `gorm:"type:varchar(36);not null;index"   json:"session_id"`
	UserID       string    `gorm:"type:varchar(36);not null"        json:"user_id"`
	SceneID      string    `gorm:"type:varchar(50);not null"        json:"scene_id"`
	StepID       int       `gorm:"type:int;not null"                json:"step_id"`
	Attempt      int       `gorm:"type:int;not null;default:1"      json:"attempt"`
	UserText     string    `gorm:"type:text"                        json:"user_text"`
	UserAudioURL string    `gorm:"type:varchar(500)"                json:"user_audio_url"`
	MatchResult  string    `gorm:"type:varchar(20);not null"        json:"match_result"`
	AIReplyText  string    `gorm:"type:text"                        json:"ai_reply_text"`
	AIAudioURL   string    `gorm:"type:varchar(500)"                json:"ai_audio_url"`
	LLMStatus    string    `gorm:"type:varchar(10)"                 json:"llm_status"`
	StepAdvanced bool      `gorm:"type:boolean;default:false"       json:"step_advanced"`
	CreatedAt    time.Time `gorm:"autoCreateTime"                   json:"created_at"`
}

// TableName 表名
func (SceneMessage) TableName() string { return "scene_messages" }
