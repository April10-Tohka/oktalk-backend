// Package model 定义发音评测相关数据模型
package model

import (
	"time"
)

// PronunciationEvaluation 发音评测记录表
// 存储用户的发音评测结果和分级反馈数据
// 对应数据库表: pronunciation_evaluations
//
// 分级反馈机制：
//   - S 级 (90-100): 纯鼓励，不提供示范音频
//   - A 级 (70-89):  鼓励+诊断，提供问题单词示范音频
//   - B 级 (50-69):  诊断+示范，提供问题单词示范音频
//   - C 级 (0-49):   完整示范，提供整句示范音频
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

	// === 评分字段 ===
	// OverallScore 综合评分（0-100）
	OverallScore int `gorm:"type:int;default:0;not null" json:"overall_score" validate:"gte=0,lte=100"`
	// AccuracyScore 准确度评分（0-100）
	AccuracyScore int `gorm:"type:int;default:0;not null" json:"accuracy_score" validate:"gte=0,lte=100"`
	// FluencyScore 流利度评分（0-100）
	FluencyScore int `gorm:"type:int;default:0;not null" json:"fluency_score" validate:"gte=0,lte=100"`
	// IntegrityScore 完整度评分（0-100）
	IntegrityScore int `gorm:"type:int;default:0;not null" json:"integrity_score" validate:"gte=0,lte=100"`

	// === 反馈字段 ===
	// FeedbackLevel 反馈级别：S/A/B/C（根据 overall_score 计算）
	FeedbackLevel string `gorm:"index;type:enum('S','A','B','C');default:'C';not null" json:"feedback_level" validate:"required,oneof=S A B C"`
	// FeedbackText 反馈文本内容（由 LLM 生成）
	FeedbackText *string `gorm:"type:text" json:"feedback_text,omitempty"`
	// FeedbackAudioURL 反馈音频 URL（由 TTS 生成）
	FeedbackAudioURL *string `gorm:"type:varchar(500)" json:"feedback_audio_url,omitempty" validate:"omitempty,url,max=500"`

	// === 问题单词字段（A/B 级使用）===
	// ProblemWords 问题单词列表 (JSON 数组: ["apple", "oranges"])
	ProblemWords StringArray `gorm:"type:json" json:"problem_words,omitempty"`
	// ProblemWordAudioURLs 问题单词示范音频 URL (JSON 对象: {"apple": "url1", "oranges": "url2"})
	ProblemWordAudioURLs StringMap `gorm:"type:json" json:"problem_word_audio_urls,omitempty"`

	// === 整句示范字段（C 级使用）===
	// DemoSentenceAudioURL 整句示范音频 URL（仅 C 级需要）
	DemoSentenceAudioURL *string `gorm:"type:varchar(500)" json:"demo_sentence_audio_url,omitempty" validate:"omitempty,url,max=500"`

	// === 其他字段 ===
	// DifficultyLevel 难度级别：beginner/intermediate/advanced
	DifficultyLevel string `gorm:"type:enum('beginner','intermediate','advanced');default:'beginner';not null" json:"difficulty_level" validate:"required,oneof=beginner intermediate advanced"`
	// AssessmentSID 语音评测会话 ID（例如讯飞返回的 SID）
	AssessmentSID *string `gorm:"type:varchar(100)" json:"assessment_sid,omitempty" validate:"omitempty,max=100"`
	// SpeechAssessmentJSON 语音评测服务返回的原始评测数据（JSON）
	SpeechAssessmentJSON *string `gorm:"type:longtext" json:"speech_assessment_json,omitempty"`
	// Status 状态：pending/processing/completed/failed
	Status string `gorm:"index;type:enum('pending','processing','completed','failed');default:'pending';not null" json:"status" validate:"required,oneof=pending processing completed failed"`
	// ErrorMessage 错误信息（失败时）
	ErrorMessage *string `gorm:"type:varchar(500)" json:"error_message,omitempty" validate:"omitempty,max=500"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName 指定表名
func (PronunciationEvaluation) TableName() string {
	return "pronunciation_evaluations"
}
