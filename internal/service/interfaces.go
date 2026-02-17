// Package service 定义所有服务接口
package service

import (
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
	//Generate(ctx context.Context, eval *model.PronunciationEvaluation) (*model.Feedback, error)
}

// UserService 用户服务接口
type UserService interface {
	user.Service
}

// AudioService 音频服务接口
type AudioService interface {
	//Process(ctx context.Context, data []byte, format string) ([]byte, error)
	//Upload(ctx context.Context, data []byte, filename string) (string, error)
}

// ReportService 报告服务接口
type ReportService interface {
	//GenerateWeeklyReport(ctx context.Context, userID string) (*model.LearningReport, error)
	//GenerateMonthlyReport(ctx context.Context, userID string, year, month int) (*model.LearningReport, error)
	//GetReport(ctx context.Context, reportID string) (*model.LearningReport, error)
}

// ConversationService 对话服务接口
type ConversationService interface {
	//CreateConversation(ctx context.Context, userID string, topic string) (*model.VoiceConversation, error)
	//SendMessage(ctx context.Context, conversationID string, message string) (*model.ConversationMessage, error)
	//GetHistory(ctx context.Context, conversationID string) ([]*model.ConversationMessage, error)
}

// Services 服务集合
type Services struct {
	Evaluation   EvaluationService
	Feedback     FeedbackService
	User         UserService
	Audio        AudioService
	Report       ReportService
	Conversation ConversationService
}

// FeedbackServiceInterface feedback 服务接口定义
type FeedbackServiceInterface interface {
	feedback.Service
}
