// Package model 定义学习报告相关数据模型
package model

import (
	"time"
)

// LearningReport 学习报告表
// 存储用户的学习报告（周报/月报），支持雷达图分析
// 对应数据库表: learning_reports
//
// 雷达图三维分析：
//   - 准确度（Accuracy）: average_accuracy_score
//   - 流利度（Fluency）:  average_fluency_score
//   - 完整度（Integrity）: average_integrity_score
type LearningReport struct {
	// ID 报告 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// UserID 用户 ID，外键
	UserID string `gorm:"index;type:varchar(36);not null" json:"user_id" validate:"required,uuid"`
	// ReportType 报告类型：weekly/monthly/custom
	ReportType string `gorm:"index;type:enum('weekly','monthly','custom');default:'weekly';not null" json:"report_type" validate:"required,oneof=weekly monthly custom"`
	// PeriodStartDate 统计周期起始日期
	PeriodStartDate time.Time `gorm:"type:date;not null;index" json:"period_start_date" validate:"required"`
	// PeriodEndDate 统计周期结束日期
	PeriodEndDate time.Time `gorm:"type:date;not null" json:"period_end_date" validate:"required"`

	// === 统计数据 ===
	// TotalConversations 总对话数
	TotalConversations int `gorm:"type:int;default:0;not null" json:"total_conversations" validate:"gte=0"`
	// TotalEvaluations 总评测数
	TotalEvaluations int `gorm:"type:int;default:0;not null" json:"total_evaluations" validate:"gte=0"`
	// TotalStudyMinutes 总学习时长（分钟）
	TotalStudyMinutes int `gorm:"type:int;default:0;not null" json:"total_study_minutes" validate:"gte=0"`

	// === 平均分数 ===
	// AverageConversationScore 平均对话评分
	AverageConversationScore float64 `gorm:"type:float;default:0.0;not null" json:"average_conversation_score" validate:"gte=0,lte=100"`
	// AverageEvaluationScore 平均评测分数
	AverageEvaluationScore float64 `gorm:"type:float;default:0.0;not null" json:"average_evaluation_score" validate:"gte=0,lte=100"`

	// === 雷达图数据 ===
	// AverageAccuracyScore 平均准确度评分（用于雷达图）
	AverageAccuracyScore float64 `gorm:"type:float;default:0.0;not null" json:"average_accuracy_score" validate:"gte=0,lte=100"`
	// AverageFluencyScore 平均流利度评分（用于雷达图）
	AverageFluencyScore float64 `gorm:"type:float;default:0.0;not null" json:"average_fluency_score" validate:"gte=0,lte=100"`
	// AverageIntegrityScore 平均完整度评分（用于雷达图）
	AverageIntegrityScore float64 `gorm:"type:float;default:0.0;not null" json:"average_integrity_score" validate:"gte=0,lte=100"`

	// === 分级统计 ===
	// SLevelCount S 级评测数
	SLevelCount int `gorm:"type:int;default:0;not null" json:"s_level_count" validate:"gte=0"`
	// ALevelCount A 级评测数
	ALevelCount int `gorm:"type:int;default:0;not null" json:"a_level_count" validate:"gte=0"`
	// BLevelCount B 级评测数
	BLevelCount int `gorm:"type:int;default:0;not null" json:"b_level_count" validate:"gte=0"`
	// CLevelCount C 级评测数
	CLevelCount int `gorm:"type:int;default:0;not null" json:"c_level_count" validate:"gte=0"`

	// === 进步分析 ===
	// ImprovementRate 进步率（与上期对比，百分比）
	ImprovementRate float64 `gorm:"type:float;default:0.0;not null" json:"improvement_rate"`

	// === JSON 分析字段 ===
	// MostPracticedTopics 最常练习的主题（JSON 数组: ["Greetings", "Daily Routine"]）
	MostPracticedTopics StringArray `gorm:"type:json" json:"most_practiced_topics,omitempty"`
	// ProblemWords 高频问题单词（JSON 数组: ["apple", "orange", "beautiful"]）
	ProblemWords StringArray `gorm:"type:json" json:"problem_words,omitempty"`
	// Strengths 优势分析（JSON 数组: ["流利度提升明显", "完整度优秀"]）
	Strengths StringArray `gorm:"type:json" json:"strengths,omitempty"`
	// Weaknesses 不足分析（JSON 数组: ["准确度需加强", "部分单词发音不清晰"]）
	Weaknesses StringArray `gorm:"type:json" json:"weaknesses,omitempty"`

	// === AI 建议 ===
	// Recommendations AI 生成的学习建议
	Recommendations *string `gorm:"type:longtext" json:"recommendations,omitempty"`
	// AIModel 生成学习建议使用的 AI 模型
	AIModel *string `gorm:"type:varchar(100)" json:"ai_model,omitempty" validate:"omitempty,max=100"`

	// === 时间戳 ===
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName 指定表名
func (LearningReport) TableName() string {
	return "learning_reports"
}
