// Package evaluation 定义评测服务接口
package evaluation

// Service 评测服务接口
type Service interface {
	// Submit 提交发音评测
	//Submit(ctx context.Context, req *EvaluationRequest) (*model.Evaluation, error)

	// GetResult 获取评测结果
	//GetResult(ctx context.Context, evaluationID string) (*model.Evaluation, error)

	// GetHistory 获取评测历史
	//GetHistory(ctx context.Context, userID string, page, pageSize int) ([]*model.Evaluation, int64, error)
}

// ScorerInterface 评分器接口
type ScorerInterface interface {
	// CalculateOverallScore 计算综合评分
	CalculateOverallScore(result *RawEvaluationResult) float64

	// GetLevel 根据分数获取等级
	GetLevel(score float64) string
}

// AnalyzerInterface 分析器接口
type AnalyzerInterface interface {
	// AnalyzePhonemes 分析音素级发音问题
	AnalyzePhonemes(result *RawEvaluationResult) *PhonemeAnalysis

	// AnalyzeProblems 分析发音问题
	AnalyzeProblems(result *RawEvaluationResult) []*PronunciationProblem
}
