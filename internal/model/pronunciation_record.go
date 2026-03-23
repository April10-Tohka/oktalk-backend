package model

import "time"

// PronunciationRecord 发音练习 v2 单次评测记录
type PronunciationRecord struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"       json:"id"`
	SessionID    string    `gorm:"type:varchar(36);not null;index"   json:"session_id"`
	UserID       string    `gorm:"type:varchar(36);not null"         json:"user_id"`
	UnitID       string    `gorm:"type:varchar(50);not null"         json:"unit_id"`
	ItemID       int       `gorm:"type:int;not null"                 json:"item_id"`
	Content      string    `gorm:"type:varchar(500);not null"        json:"content"`
	PracticeType string    `gorm:"type:varchar(20);not null"         json:"practice_type"`
	RawScore     float32   `gorm:"type:decimal(3,1);not null"        json:"raw_score"`
	Stars        int       `gorm:"type:int;not null"                 json:"stars"`
	ProblemWords string    `gorm:"type:json"                         json:"problem_words"`
	UserAudioURL string    `gorm:"type:varchar(500)"                 json:"user_audio_url"`
	AIEncourage  string    `gorm:"type:text"                         json:"ai_encourage"`
	AIProblemTip string    `gorm:"type:text"                         json:"ai_problem_tip"`
	AISuggestion string    `gorm:"type:text"                         json:"ai_suggestion"`
	AIAudioURL   string    `gorm:"type:varchar(500)"                 json:"ai_audio_url"`
	// 科大讯飞细分评测指标（0.0-5.0），对应 domain.EvaluationResult
	Fluency    float32 `gorm:"type:decimal(3,1);default:0" json:"fluency"`      // FluencyScore
	Integrity  float32 `gorm:"type:decimal(3,1);default:0" json:"integrity"`   // IntegrityScore
	PhoneScore float32 `gorm:"type:decimal(3,1);default:0" json:"phone_score"` // StandardScore（标准度/音素相关）
	CreatedAt  time.Time `gorm:"autoCreateTime"                    json:"created_at"`
}

// TableName 表名
func (PronunciationRecord) TableName() string { return "pronunciation_records" }
