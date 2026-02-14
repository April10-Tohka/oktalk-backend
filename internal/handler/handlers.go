// Package handler 提供 Handler 聚合结构
// 用于依赖注入时统一传递所有 Handler
package handler

// Handlers 所有 HTTP Handler 的聚合
type Handlers struct {
	Health     *HealthHandler
	User       *UserHandler
	Evaluation *EvaluationHandler
	Feedback   *FeedbackHandler
}
