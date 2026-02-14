// Package user 提供用户评测记录管理
package user

import (
	"context"
	"time"
)

// Recorder 用户评测记录管理器
type Recorder struct {
	// TODO: 添加依赖
}

// NewRecorder 创建记录管理器
func NewRecorder() *Recorder {
	return &Recorder{}
}

// RecordEvaluation 记录评测
func (r *Recorder) RecordEvaluation(ctx context.Context, record *EvaluationRecord) error {
	// TODO: 实现评测记录
	return nil
}

// GetDailyRecords 获取每日记录
func (r *Recorder) GetDailyRecords(ctx context.Context, userID string, date time.Time) ([]*EvaluationRecord, error) {
	// TODO: 实现获取每日记录
	return nil, nil
}

// GetWeeklyProgress 获取周进度
func (r *Recorder) GetWeeklyProgress(ctx context.Context, userID string) (*WeeklyProgress, error) {
	// TODO: 实现获取周进度
	return nil, nil
}

// GetMonthlyReport 获取月度报告数据
func (r *Recorder) GetMonthlyReport(ctx context.Context, userID string, year, month int) (*MonthlyReport, error) {
	// TODO: 实现获取月度报告
	return nil, nil
}

// EvaluationRecord 评测记录
type EvaluationRecord struct {
	UserID       string    `json:"user_id"`
	EvaluationID string    `json:"evaluation_id"`
	TextID       string    `json:"text_id"`
	Score        float64   `json:"score"`
	Duration     int       `json:"duration"`
	CreatedAt    time.Time `json:"created_at"`
}

// WeeklyProgress 周进度
type WeeklyProgress struct {
	UserID          string    `json:"user_id"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	TotalCount      int       `json:"total_count"`
	AverageScore    float64   `json:"average_score"`
	DailyCount      []int     `json:"daily_count"`
	DailyAvgScores  []float64 `json:"daily_avg_scores"`
	Improvement     float64   `json:"improvement"`
}

// MonthlyReport 月度报告
type MonthlyReport struct {
	UserID           string              `json:"user_id"`
	Year             int                 `json:"year"`
	Month            int                 `json:"month"`
	TotalEvaluations int                 `json:"total_evaluations"`
	AverageScore     float64             `json:"average_score"`
	BestScore        float64             `json:"best_score"`
	CommonErrors     []string            `json:"common_errors"`
	Improvements     []string            `json:"improvements"`
	Recommendations  []string            `json:"recommendations"`
}
