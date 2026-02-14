// Package model 定义通用数据结构和辅助方法
package model

import (
	"time"
)

// Pagination 分页参数
type Pagination struct {
	// Page 当前页码（从 1 开始）
	Page int `json:"page" form:"page" validate:"gte=1"`
	// PageSize 每页数量
	PageSize int `json:"page_size" form:"page_size" validate:"gte=1,lte=100"`
}

// GetOffset 计算偏移量
func (p *Pagination) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return (p.Page - 1) * p.PageSize
}

// Normalize 规范化分页参数
func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// PaginatedResult 分页结果
type PaginatedResult struct {
	// Items 数据列表
	Items interface{} `json:"items"`
	// Total 总记录数
	Total int64 `json:"total"`
	// Page 当前页码
	Page int `json:"page"`
	// PageSize 每页数量
	PageSize int `json:"page_size"`
	// TotalPages 总页数
	TotalPages int `json:"total_pages"`
}

// NewPaginatedResult 创建分页结果
func NewPaginatedResult(items interface{}, total int64, page, pageSize int) *PaginatedResult {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &PaginatedResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// APIResponse 统一 API 响应格式
type APIResponse struct {
	// Code 状态码，0 表示成功
	Code int `json:"code"`
	// Message 响应消息
	Message string `json:"message"`
	// Data 响应数据
	Data interface{} `json:"data,omitempty"`
	// Timestamp 响应时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *APIResponse {
	return &APIResponse{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
}

// NewPaginatedResponse 创建分页响应
func NewPaginatedResponse(items interface{}, total int64, page, pageSize int) *APIResponse {
	return &APIResponse{
		Code:      0,
		Message:   "success",
		Data:      NewPaginatedResult(items, total, page, pageSize),
		Timestamp: time.Now().Unix(),
	}
}

// 用户状态常量
const (
	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"
	UserStatusDeleted   = "deleted"
)

// 反馈级别常量
const (
	FeedbackLevelS = "S"
	FeedbackLevelA = "A"
	FeedbackLevelB = "B"
	FeedbackLevelC = "C"
)

// 难度级别常量
const (
	DifficultyBeginner     = "beginner"
	DifficultyIntermediate = "intermediate"
	DifficultyAdvanced     = "advanced"
)

// 对话状态常量
const (
	ConversationStatusActive    = "active"
	ConversationStatusCompleted = "completed"
	ConversationStatusPaused    = "paused"
)

// 评测状态常量
const (
	EvaluationStatusPending    = "pending"
	EvaluationStatusProcessing = "processing"
	EvaluationStatusCompleted  = "completed"
	EvaluationStatusFailed     = "failed"
)

// 发送者类型常量
const (
	SenderTypeUser = "user"
	SenderTypeAI   = "ai"
)

// 报告类型常量
const (
	ReportTypeWeekly  = "weekly"
	ReportTypeMonthly = "monthly"
	ReportTypeCustom  = "custom"
)

// Evaluation 评测模型别名（方便引用）
type Evaluation = PronunciationEvaluation

// Feedback 反馈模型
// 用于服务层传递反馈数据
type Feedback struct {
	ID           string `json:"id"`
	EvaluationID string `json:"evaluation_id"`
	Level        string `json:"level"`
	Text         string `json:"text"`
	AudioURL     string `json:"audio_url,omitempty"`
	Suggestions  string `json:"suggestions,omitempty"`
}

// GetFeedbackLevel 根据分数获取反馈级别
func GetFeedbackLevel(score int) string {
	switch {
	case score >= 90:
		return FeedbackLevelS
	case score >= 70:
		return FeedbackLevelA
	case score >= 50:
		return FeedbackLevelB
	default:
		return FeedbackLevelC
	}
}
