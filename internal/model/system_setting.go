// Package model 定义系统配置数据模型
package model

import (
	"time"
)

// SystemSetting 系统配置表
// 存储系统级配置（评分阈值、反馈模板等）
type SystemSetting struct {
	// ID 主键 ID (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// ConfigKey 配置键，唯一
	ConfigKey string `gorm:"uniqueIndex;type:varchar(100);not null" json:"config_key" validate:"required,max=100"`
	// ConfigValue 配置值（支持 JSON）
	ConfigValue string `gorm:"type:longtext;not null" json:"config_value" validate:"required"`
	// ConfigType 配置类型：string/int/float/json/boolean
	ConfigType string `gorm:"type:enum('string','int','float','json','boolean');not null" json:"config_type" validate:"required,oneof=string int float json boolean"`
	// Description 配置描述
	Description *string `gorm:"type:varchar(500)" json:"description,omitempty" validate:"omitempty,max=500"`
	// IsEditable 是否可编辑
	IsEditable bool `gorm:"type:boolean;default:true;not null" json:"is_editable"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}

// 系统配置键的常量定义
const (
	// ConfigFeedbackSLevelMinScore S 级反馈最低分数
	ConfigFeedbackSLevelMinScore = "feedback_s_level_min_score"
	// ConfigFeedbackALevelMinScore A 级反馈最低分数
	ConfigFeedbackALevelMinScore = "feedback_a_level_min_score"
	// ConfigFeedbackBLevelMinScore B 级反馈最低分数
	ConfigFeedbackBLevelMinScore = "feedback_b_level_min_score"
	// ConfigFeedbackCLevelMinScore C 级反馈最低分数
	ConfigFeedbackCLevelMinScore = "feedback_c_level_min_score"
	// ConfigForbiddenWords 禁用词列表
	ConfigForbiddenWords = "forbidden_words"
	// ConfigReportGenerationSchedule 报告生成时间表 (cron)
	ConfigReportGenerationSchedule = "report_generation_schedule"
)

// DefaultSystemSettings 默认系统配置
var DefaultSystemSettings = []SystemSetting{
	{
		ID:          "set_001",
		ConfigKey:   ConfigFeedbackSLevelMinScore,
		ConfigValue: "90",
		ConfigType:  "int",
		Description: strPtr("S级反馈最低分数"),
		IsEditable:  true,
	},
	{
		ID:          "set_002",
		ConfigKey:   ConfigFeedbackALevelMinScore,
		ConfigValue: "70",
		ConfigType:  "int",
		Description: strPtr("A级反馈最低分数"),
		IsEditable:  true,
	},
	{
		ID:          "set_003",
		ConfigKey:   ConfigFeedbackBLevelMinScore,
		ConfigValue: "50",
		ConfigType:  "int",
		Description: strPtr("B级反馈最低分数"),
		IsEditable:  true,
	},
	{
		ID:          "set_004",
		ConfigKey:   ConfigFeedbackCLevelMinScore,
		ConfigValue: "0",
		ConfigType:  "int",
		Description: strPtr("C级反馈最低分数"),
		IsEditable:  true,
	},
	{
		ID:          "set_005",
		ConfigKey:   ConfigForbiddenWords,
		ConfigValue: `["bad","wrong","terrible","fail","mistake","error","poor","awful"]`,
		ConfigType:  "json",
		Description: strPtr("禁用词列表"),
		IsEditable:  true,
	},
	{
		ID:          "set_006",
		ConfigKey:   ConfigReportGenerationSchedule,
		ConfigValue: "0 0 * * 1",
		ConfigType:  "string",
		Description: strPtr("报告生成时间表(cron)"),
		IsEditable:  true,
	},
}

// strPtr 字符串指针辅助函数
func strPtr(s string) *string {
	return &s
}
