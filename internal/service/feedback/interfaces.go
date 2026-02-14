// Package feedback 定义反馈服务接口
package feedback

// Service 反馈服务接口
type Service interface {
	// GetFeedback 获取反馈
	//GetFeedback(ctx context.Context, evaluationID string) (*model.Feedback, error)

	// GenerateFeedback 生成反馈
	//GenerateFeedback(ctx context.Context, evaluation *model.Evaluation) (*model.Feedback, error)
}
