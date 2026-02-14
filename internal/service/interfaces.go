// Package service 定义所有服务接口
package service

import (
	"context"

	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/service/evaluation"
	"pronunciation-correction-system/internal/service/feedback"
	"pronunciation-correction-system/internal/service/user"
)

// EvaluationService 评测服务接口
type EvaluationService interface {
	evaluation.Service
}

// FeedbackService 反馈服务接口
type FeedbackService interface {
	//Generate(ctx context.Context, eval *model.Evaluation) (*model.Feedback, error)
}

// UserService 用户服务接口
type UserService interface {
	user.Service
}

// AudioService 音频服务接口
type AudioService interface {
	Process(ctx context.Context, data []byte, format string) ([]byte, error)
	Upload(ctx context.Context, data []byte, filename string) (string, error)
}

// ReportService 报告服务接口
type ReportService interface {
	//GenerateWeeklyReport(ctx context.Context, userID string) (*model.Report, error)
	//GenerateMonthlyReport(ctx context.Context, userID string, year, month int) (*model.Report, error)
	//GetReport(ctx context.Context, reportID string) (*model.Report, error)
}

// ChatService 对话服务接口
type ChatService interface {
	CreateSession(ctx context.Context, userID string, scenario string) (*model.ChatSession, error)
	SendMessage(ctx context.Context, sessionID string, message string) (*model.ChatMessage, error)
	GetHistory(ctx context.Context, sessionID string) ([]*model.ChatMessage, error)
}

// Services 服务集合
type Services struct {
	Evaluation EvaluationService
	Feedback   FeedbackService
	User       UserService
	Audio      AudioService
	Report     ReportService
	Chat       ChatService
}

// FeedbackServiceInterface feedback 服务接口定义
type FeedbackServiceInterface interface {
	feedback.Service
}
