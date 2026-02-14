// Package model 定义反馈记录数据模型
package model

import (
	"time"
)

// FeedbackRecord 反馈记录表
// 存储系统生成的反馈记录（用于反馈质量追踪）
type FeedbackRecord struct {
	// ID 反馈 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// EvaluationID 关联的评测 ID，外键，唯一
	EvaluationID string `gorm:"uniqueIndex;type:varchar(36);not null" json:"evaluation_id" validate:"required,uuid"`
	// FeedbackLevel 反馈级别：S/A/B/C
	FeedbackLevel string `gorm:"index;type:enum('S','A','B','C');not null" json:"feedback_level" validate:"required,oneof=S A B C"`
	// FeedbackText 反馈文本
	FeedbackText string `gorm:"type:text;not null" json:"feedback_text" validate:"required"`
	// FeedbackAudioURL 反馈音频 URL
	FeedbackAudioURL *string `gorm:"type:varchar(500)" json:"feedback_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// DemoType 示范类型：word/sentence/NULL
	DemoType *string `gorm:"type:enum('word','sentence')" json:"demo_type,omitempty" validate:"omitempty,oneof=word sentence"`
	// DemoContent 示范内容
	DemoContent *string `gorm:"type:varchar(500)" json:"demo_content,omitempty" validate:"omitempty,max=500"`
	// DemoAudioURL 示范音频 URL
	DemoAudioURL *string `gorm:"type:varchar(500)" json:"demo_audio_url,omitempty" validate:"omitempty,url,max=500"`
	// GenerationMethod 生成方法：llm/template/manual
	GenerationMethod string `gorm:"type:varchar(50);default:'llm';not null" json:"generation_method" validate:"required,oneof=llm template manual"`
	// AIModel 使用的 AI 模型（如 qwen-max）
	AIModel *string `gorm:"type:varchar(100)" json:"ai_model,omitempty" validate:"omitempty,max=100"`
	// GenerationDurationMS 生成耗时（毫秒）
	GenerationDurationMS *int `gorm:"type:int" json:"generation_duration_ms,omitempty" validate:"omitempty,gte=0"`
	// Status 状态：generating/completed/failed
	Status string `gorm:"index;type:enum('generating','completed','failed');default:'completed';not null" json:"status" validate:"required,oneof=generating completed failed"`
	// QualityScore 反馈质量评分（0-100，用于内部评估）
	QualityScore *int `gorm:"type:int" json:"quality_score,omitempty" validate:"omitempty,min=0,max=100"`
	// UserFeedback 用户对反馈的评价
	UserFeedback *string `gorm:"type:varchar(500)" json:"user_feedback,omitempty" validate:"omitempty,max=500"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	Evaluation *PronunciationEvaluation `gorm:"foreignKey:EvaluationID;references:ID" json:"evaluation,omitempty"`
}

// TableName 指定表名
func (FeedbackRecord) TableName() string {
	return "feedback_records"
}
