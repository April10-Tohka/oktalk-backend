// Package model 定义学习报告相关数据模型
package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// LearningReport 学习报告表
// 存储用户的学习报告（周报/月报）
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
	// TotalConversations 总对话数
	TotalConversations int `gorm:"type:int;default:0;not null" json:"total_conversations" validate:"gte=0"`
	// TotalEvaluations 总评测数
	TotalEvaluations int `gorm:"type:int;default:0;not null" json:"total_evaluations" validate:"gte=0"`
	// TotalStudyMinutes 总学习时长（分钟）
	TotalStudyMinutes int `gorm:"type:int;default:0;not null" json:"total_study_minutes" validate:"gte=0"`
	// AverageConversationScore 平均对话评分
	AverageConversationScore float64 `gorm:"type:float;default:0.0;not null" json:"average_conversation_score" validate:"gte=0,lte=100"`
	// AverageEvaluationScore 平均评测分数
	AverageEvaluationScore float64 `gorm:"type:float;default:0.0;not null" json:"average_evaluation_score" validate:"gte=0,lte=100"`
	// SLevelCount S 级评测数
	SLevelCount int `gorm:"type:int;default:0;not null" json:"s_level_count" validate:"gte=0"`
	// ALevelCount A 级评测数
	ALevelCount int `gorm:"type:int;default:0;not null" json:"a_level_count" validate:"gte=0"`
	// BLevelCount B 级评测数
	BLevelCount int `gorm:"type:int;default:0;not null" json:"b_level_count" validate:"gte=0"`
	// CLevelCount C 级评测数
	CLevelCount int `gorm:"type:int;default:0;not null" json:"c_level_count" validate:"gte=0"`
	// ImprovementRate 进步率（与上期对比，%）
	ImprovementRate float64 `gorm:"type:float;default:0.0;not null" json:"improvement_rate"`
	// Strengths 优势总结（JSON 数组）
	Strengths StringArray `gorm:"type:json" json:"strengths,omitempty"`
	// Weaknesses 不足总结（JSON 数组）
	Weaknesses StringArray `gorm:"type:json" json:"weaknesses,omitempty"`
	// Recommendations 改进建议
	Recommendations *string `gorm:"type:longtext" json:"recommendations,omitempty"`
	// ReportContent 完整报告内容（HTML/Markdown）
	ReportContent *string `gorm:"type:longtext" json:"report_content,omitempty"`
	// GeneratedBy 生成者：system/manual
	GeneratedBy string `gorm:"type:varchar(50);default:'system';not null" json:"generated_by" validate:"required,oneof=system manual"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"index;autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	User       *User              `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Statistics []*ReportStatistic `gorm:"foreignKey:ReportID" json:"statistics,omitempty"`
}

// TableName 指定表名
func (LearningReport) TableName() string {
	return "learning_reports"
}

// ReportStatistic 报告统计明细表
// 存储报告中的具体统计数据（便于细粒度分析）
type ReportStatistic struct {
	// ID 主键 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// ReportID 报告 ID，外键
	ReportID string `gorm:"index;type:varchar(36);not null" json:"report_id" validate:"required,uuid"`
	// StatDate 统计日期
	StatDate time.Time `gorm:"index;type:date;not null" json:"stat_date" validate:"required"`
	// DailyConversations 当日对话数
	DailyConversations int `gorm:"type:int;default:0;not null" json:"daily_conversations" validate:"gte=0"`
	// DailyEvaluations 当日评测数
	DailyEvaluations int `gorm:"type:int;default:0;not null" json:"daily_evaluations" validate:"gte=0"`
	// DailyStudyMinutes 当日学习时长（分钟）
	DailyStudyMinutes int `gorm:"type:int;default:0;not null" json:"daily_study_minutes" validate:"gte=0"`
	// DailyAvgEvalScore 当日平均评测分数
	DailyAvgEvalScore float64 `gorm:"type:float;default:0.0;not null" json:"daily_avg_eval_score" validate:"gte=0,lte=100"`
	// DailySLevelCount 当日 S 级数
	DailySLevelCount int `gorm:"type:int;default:0;not null" json:"daily_s_level_count" validate:"gte=0"`
	// DailyALevelCount 当日 A 级数
	DailyALevelCount int `gorm:"type:int;default:0;not null" json:"daily_a_level_count" validate:"gte=0"`
	// DailyBLevelCount 当日 B 级数
	DailyBLevelCount int `gorm:"type:int;default:0;not null" json:"daily_b_level_count" validate:"gte=0"`
	// DailyCLevelCount 当日 C 级数
	DailyCLevelCount int `gorm:"type:int;default:0;not null" json:"daily_c_level_count" validate:"gte=0"`
	// TopicBreakdown 主题分布（JSON）：{topic: count}
	TopicBreakdown TopicBreakdownMap `gorm:"type:json" json:"topic_breakdown,omitempty"`
	// DifficultyBreakdown 难度分布（JSON）：{level: count}
	DifficultyBreakdown DifficultyBreakdownMap `gorm:"type:json" json:"difficulty_breakdown,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp" json:"created_at"`

	// 关联
	Report *LearningReport `gorm:"foreignKey:ReportID;references:ID" json:"report,omitempty"`
}

// TableName 指定表名
func (ReportStatistic) TableName() string {
	return "report_statistics"
}

// StringArray 字符串数组（用于 JSON 序列化）
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, sa)
}

// Value 实现 driver.Valuer 接口
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return nil, nil
	}
	return json.Marshal(sa)
}

// TopicBreakdownMap 主题分布数据
type TopicBreakdownMap map[string]int

// Scan 实现 sql.Scanner 接口
func (tbm *TopicBreakdownMap) Scan(value interface{}) error {
	if value == nil {
		*tbm = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, tbm)
}

// Value 实现 driver.Valuer 接口
func (tbm TopicBreakdownMap) Value() (driver.Value, error) {
	if tbm == nil {
		return nil, nil
	}
	return json.Marshal(tbm)
}

// DifficultyBreakdownMap 难度分布数据
type DifficultyBreakdownMap map[string]int

// Scan 实现 sql.Scanner 接口
func (dbm *DifficultyBreakdownMap) Scan(value interface{}) error {
	if value == nil {
		*dbm = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, dbm)
}

// Value 实现 driver.Valuer 接口
func (dbm DifficultyBreakdownMap) Value() (driver.Value, error) {
	if dbm == nil {
		return nil, nil
	}
	return json.Marshal(dbm)
}
