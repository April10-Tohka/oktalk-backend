// Package model 定义发音评测相关数据模型
package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// PronunciationEvaluation 发音评测记录表
// 存储用户的发音评测结果（核心数据）
type PronunciationEvaluation struct {
	// ID 评测 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// UserID 用户 ID，外键
	UserID string `gorm:"index;type:varchar(36);not null" json:"user_id" validate:"required,uuid"`
	// TargetText 目标朗读文本
	TargetText string `gorm:"type:varchar(500);not null" json:"target_text" validate:"required,max=500"`
	// RecognizedText 识别出的文本
	RecognizedText *string `gorm:"type:varchar(500)" json:"recognized_text,omitempty" validate:"omitempty,max=500"`
	// AudioURL 原始录音 URL
	AudioURL *string `gorm:"type:varchar(500)" json:"audio_url,omitempty" validate:"omitempty,url,max=500"`
	// AudioDuration 录音时长（秒）
	AudioDuration *int `gorm:"type:int" json:"audio_duration,omitempty" validate:"omitempty,gte=0"`
	// OverallScore 综合评分（0-100）
	OverallScore int `gorm:"type:int;default:0;not null" json:"overall_score" validate:"gte=0,lte=100"`
	// AccuracyScore 准确度评分（0-100）
	AccuracyScore int `gorm:"type:int;default:0;not null" json:"accuracy_score" validate:"gte=0,lte=100"`
	// FluencyScore 流利度评分（0-100）
	FluencyScore int `gorm:"type:int;default:0;not null" json:"fluency_score" validate:"gte=0,lte=100"`
	// IntegrityScore 完整度评分（0-100）
	IntegrityScore int `gorm:"type:int;default:0;not null" json:"integrity_score" validate:"gte=0,lte=100"`
	// FeedbackLevel 反馈级别：S/A/B/C
	FeedbackLevel string `gorm:"index;type:enum('S','A','B','C');default:'C';not null" json:"feedback_level" validate:"required,oneof=S A B C"`
	// FeedbackText 反馈文本
	FeedbackText *string `gorm:"type:text" json:"feedback_text,omitempty"`
	// FeedbackAudioURL 反馈音频 URL
	FeedbackAudioURL *string `gorm:"type:varchar(500)" json:"feedback_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// DemoAudioType 示范音频类型：word/sentence/NULL
	DemoAudioType *string `gorm:"type:enum('word','sentence')" json:"demo_audio_type,omitempty" validate:"omitempty,oneof=word sentence"`
	// DemoAudioContent 示范内容（如 "apples" 或完整句子）
	DemoAudioContent *string `gorm:"type:varchar(500)" json:"demo_audio_content,omitempty" validate:"omitempty,max=500"`
	// DemoAudioURL 示范音频 URL
	DemoAudioURL *string `gorm:"type:varchar(500)" json:"demo_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// DifficultyLevel 难度级别：beginner/intermediate/advanced
	DifficultyLevel string `gorm:"type:enum('beginner','intermediate','advanced');default:'beginner';not null" json:"difficulty_level" validate:"required,oneof=beginner intermediate advanced"`
	// SpeechAssessmentJSON 讯飞返回的原始评测数据（JSON）
	SpeechAssessmentJSON *string `gorm:"type:longtext" json:"speech_assessment_json,omitempty"`
	// Status 状态：pending/processing/completed/failed
	Status string `gorm:"index;type:enum('pending','processing','completed','failed');default:'completed';not null" json:"status" validate:"required,oneof=pending processing completed failed"`
	// ErrorMessage 错误信息（失败时）
	ErrorMessage *string `gorm:"type:varchar(500)" json:"error_message,omitempty" validate:"omitempty,max=500"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	User              *User              `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	EvaluationDetails []*EvaluationDetail `gorm:"foreignKey:EvaluationID" json:"evaluation_details,omitempty"`
	FeedbackRecord    *FeedbackRecord    `gorm:"foreignKey:EvaluationID" json:"feedback_record,omitempty"`
}

// TableName 指定表名
func (PronunciationEvaluation) TableName() string {
	return "pronunciation_evaluations"
}

// EvaluationDetail 评测详细数据表
// 存储评测的音素级别详细数据（用于诊断和分析）
type EvaluationDetail struct {
	// ID 主键 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// EvaluationID 评测 ID，外键
	EvaluationID string `gorm:"index;type:varchar(36);not null" json:"evaluation_id" validate:"required,uuid"`
	// WordIndex 单词序号（0 开始）
	WordIndex int `gorm:"type:int;not null" json:"word_index" validate:"gte=0"`
	// WordText 单词文本
	WordText string `gorm:"type:varchar(100);not null" json:"word_text" validate:"required,max=100"`
	// WordScore 单词评分（0-100）
	WordScore int `gorm:"type:int;default:0;not null" json:"word_score" validate:"gte=0,lte=100"`
	// IsProblemWord 是否为问题单词（score < 70）
	IsProblemWord bool `gorm:"type:boolean;default:false;not null" json:"is_problem_word"`
	// PhonemeDetails 音素详情（JSON 数组）
	PhonemeDetails PhonemeDetailList `gorm:"type:json" json:"phoneme_details,omitempty"`
	// BeginTimeMS 单词开始时间（毫秒）
	BeginTimeMS *int `gorm:"type:int" json:"begin_time_ms,omitempty" validate:"omitempty,gte=0"`
	// EndTimeMS 单词结束时间（毫秒）
	EndTimeMS *int `gorm:"type:int" json:"end_time_ms,omitempty" validate:"omitempty,gte=0"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp" json:"created_at"`

	// 关联
	Evaluation *PronunciationEvaluation `gorm:"foreignKey:EvaluationID;references:ID" json:"evaluation,omitempty"`
}

// TableName 指定表名
func (EvaluationDetail) TableName() string {
	return "evaluation_details"
}

// PhonemeDetail 单个音素详情
type PhonemeDetail struct {
	// Phoneme 音素符号
	Phoneme string `json:"phoneme"`
	// Score 音素评分（0-100）
	Score int `json:"score"`
}

// PhonemeDetailList 音素详情列表（用于 JSON 序列化）
type PhonemeDetailList []PhonemeDetail

// Scan 实现 sql.Scanner 接口，用于 GORM 扫描 JSON
func (pdl *PhonemeDetailList) Scan(value interface{}) error {
	if value == nil {
		*pdl = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, pdl)
}

// Value 实现 driver.Valuer 接口，用于 GORM 写入 JSON
func (pdl PhonemeDetailList) Value() (driver.Value, error) {
	if pdl == nil {
		return nil, nil
	}
	return json.Marshal(pdl)
}
