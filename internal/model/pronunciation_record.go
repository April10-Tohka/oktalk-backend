package model

import "time"

// PronunciationRecord 发音练习 v2 单次评测记录
type PronunciationRecord struct {
	ID           string  `gorm:"primaryKey;type:varchar(36)"       json:"id"`
	SessionID    string  `gorm:"type:varchar(36);not null;index"   json:"session_id"`
	UserID       string  `gorm:"type:varchar(36);not null"         json:"user_id"`
	UnitID       string  `gorm:"type:varchar(50);not null"         json:"unit_id"`
	ItemID       int     `gorm:"type:int;not null"                 json:"item_id"`
	Content      string  `gorm:"type:varchar(500);not null"        json:"content"`
	PracticeType string  `gorm:"type:varchar(20);not null"         json:"practice_type"`
	RawScore     float64 `gorm:"type:decimal(3,1);not null"        json:"raw_score"`
	Stars        int     `gorm:"type:int;not null"                 json:"stars"`
	ProblemWords string  `gorm:"type:json"                         json:"problem_words"`
	UserAudioURL string  `gorm:"type:varchar(500)"                 json:"user_audio_url"`
	AIEncourage  string  `gorm:"type:text"                         json:"ai_encourage"`
	AIProblemTip string  `gorm:"type:text"                         json:"ai_problem_tip"`
	AIHowToFix   string  `gorm:"type:text"                         json:"ai_how_to_fix"`
	AIAudioURL   string  `gorm:"type:varchar(500)"                 json:"ai_audio_url"`
	HowToFixURL  string  `gorm:"type:varchar(500)"                 json:"how_to_fix_url"`
	// IsRejected 科大讯飞是否拒识（统计时仅统计 false）
	IsRejected bool `gorm:"type:boolean;not null;default:false;index" json:"is_rejected"`
	// 细分维度 0.0-5.0（与讯飞评测一致）
	AccuracyScore float64   `gorm:"type:decimal(3,1);not null;default:0" json:"accuracy_score"`
	Fluency       float64   `gorm:"type:decimal(3,1);not null;default:0" json:"fluency"`
	Integrity     float64   `gorm:"type:decimal(3,1);not null;default:0" json:"integrity"`
	StandardScore float64   `gorm:"type:decimal(3,1);not null;default:0" json:"standard_score"`
	CreatedAt     time.Time `gorm:"autoCreateTime"                    json:"created_at"`
}

// TableName 表名
func (PronunciationRecord) TableName() string { return "pronunciation_records" }
