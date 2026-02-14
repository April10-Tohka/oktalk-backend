// Package model 定义文本库数据模型
package model

import (
	"time"
)

// Text 练习文本模型
type Text struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	Content      string    `json:"content" gorm:"type:text;not null"`   // 文本内容
	Translation  string    `json:"translation" gorm:"type:text"`        // 中文翻译
	Phonetic     string    `json:"phonetic" gorm:"type:text"`           // 音标
	Level        string    `json:"level" gorm:"size:20;index"`          // 难度等级
	Scenario     string    `json:"scenario" gorm:"size:50;index"`       // 场景分类
	Tags         string    `json:"tags" gorm:"size:255"`                // 标签，逗号分隔
	DemoAudioURL string    `json:"demo_audio_url" gorm:"size:255"`      // 示范音频 URL
	Duration     int       `json:"duration"`                            // 示范音频时长（毫秒）
	UsageCount   int       `json:"usage_count" gorm:"default:0"`        // 使用次数
	Status       int       `json:"status" gorm:"default:1"`             // 1:启用 0:禁用
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Text) TableName() string {
	return "texts"
}

// TextLevel 文本难度等级
type TextLevel struct {
	Level       string `json:"level"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetTextLevels 获取所有难度等级
func GetTextLevels() []TextLevel {
	return []TextLevel{
		{Level: "beginner", Name: "初级", Description: "简单的单词和短语"},
		{Level: "elementary", Name: "基础", Description: "简单的句子"},
		{Level: "intermediate", Name: "中级", Description: "日常对话"},
		{Level: "upper_intermediate", Name: "中高级", Description: "复杂句型"},
		{Level: "advanced", Name: "高级", Description: "专业内容"},
	}
}

// TextScenario 文本场景
type TextScenario struct {
	Scenario    string `json:"scenario"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetTextScenarios 获取所有场景
func GetTextScenarios() []TextScenario {
	return []TextScenario{
		{Scenario: "daily", Name: "日常对话", Description: "日常生活场景"},
		{Scenario: "business", Name: "商务英语", Description: "商务工作场景"},
		{Scenario: "travel", Name: "旅行英语", Description: "旅行相关场景"},
		{Scenario: "academic", Name: "学术英语", Description: "学术研究场景"},
		{Scenario: "interview", Name: "面试英语", Description: "求职面试场景"},
	}
}
