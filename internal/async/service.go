// Package async 提供异步服务集成
// 用于简化评测服务与异步任务系统的集成
package async

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AsyncService 异步服务
// 封装 WorkerPool，提供便捷的任务提交方法
type AsyncService struct {
	pool *WorkerPool
}

// NewAsyncService 创建异步服务
func NewAsyncService(pool *WorkerPool) *AsyncService {
	return &AsyncService{pool: pool}
}

// Start 启动异步服务
func (s *AsyncService) Start() {
	s.pool.Start()
}

// Shutdown 停止异步服务
func (s *AsyncService) Shutdown(timeout time.Duration) {
	s.pool.Shutdown(timeout)
}

// SubmitFeedbackTextTask 提交 LLM 反馈文本生成任务
func (s *AsyncService) SubmitFeedbackTextTask(evaluationID string, score int, problemWord, level, targetText string) error {
	task := NewEvaluationTask(evaluationID, TaskGenerateFeedbackText, map[string]interface{}{
		DataKeyEvaluationID: evaluationID,
		DataKeyScore:        score,
		DataKeyProblemWord:  problemWord,
		DataKeyLevel:        level,
		DataKeyTargetText:   targetText,
	}).WithPriority(HighPriority).WithMaxRetries(3)

	return s.pool.Submit(task)
}

// SubmitFeedbackAudioTask 提交 TTS 反馈音频生成任务
func (s *AsyncService) SubmitFeedbackAudioTask(evaluationID, feedbackText string) error {
	task := NewEvaluationTask(evaluationID, TaskGenerateFeedbackAudio, map[string]interface{}{
		DataKeyEvaluationID:   evaluationID,
		DataKeyFeedbackText:   feedbackText,
	}).WithPriority(HighPriority).WithMaxRetries(3)

	return s.pool.Submit(task)
}

// SubmitDemoAudioTask 提交示范音频生成任务
func (s *AsyncService) SubmitDemoAudioTask(evaluationID, demoText, demoType string) error {
	task := NewEvaluationTask(evaluationID, TaskGenerateDemoAudio, map[string]interface{}{
		DataKeyEvaluationID: evaluationID,
		DataKeyDemoText:     demoText,
		DataKeyDemoType:     demoType,
	}).WithPriority(DefaultPriority).WithMaxRetries(2)

	return s.pool.Submit(task)
}

// SubmitEvaluationAsyncTasks 提交评测相关的所有异步任务
// 根据设计文档，评测完成后需要异步生成：
// 1. LLM 反馈文本
// 2. TTS 反馈音频（依赖反馈文本）
// 3. 示范音频（条件触发，评分低于90分时）
func (s *AsyncService) SubmitEvaluationAsyncTasks(ctx context.Context, params *EvaluationAsyncParams) error {
	evalID := params.EvaluationID
	score := params.OverallScore

	// 获取反馈级别
	level := GetFeedbackLevel(score)

	// 获取问题单词（取第一个问题单词）
	problemWord := ""
	if len(params.ProblemWords) > 0 {
		problemWord = params.ProblemWords[0]
	}

	// Task 1: 生成反馈文本
	if err := s.SubmitFeedbackTextTask(evalID, score, problemWord, level, params.TargetText); err != nil {
		log.Printf("[AsyncService] Failed to submit feedback text task for %s: %v", evalID, err)
		return fmt.Errorf("failed to submit feedback text task: %w", err)
	}

	// Task 2: 生成反馈语音（实际上需要等 Task 1 完成后才能执行）
	// 这里通过 Pipeline 模式或者在 Task 1 完成的回调中触发 Task 2

	// Task 3: 生成示范音频（条件触发：评分低于90分）
	if score < 90 && problemWord != "" {
		if err := s.SubmitDemoAudioTask(evalID, problemWord, DemoTypeWord); err != nil {
			log.Printf("[AsyncService] Failed to submit demo audio task for %s: %v", evalID, err)
			// 示范音频任务失败不影响主流程
		}
	}

	return nil
}

// EvaluationAsyncParams 评测异步任务参数
type EvaluationAsyncParams struct {
	EvaluationID  string   `json:"evaluation_id"`
	UserID        string   `json:"user_id"`
	TargetText    string   `json:"target_text"`
	OverallScore  int      `json:"overall_score"`
	AccuracyScore int      `json:"accuracy_score"`
	FluencyScore  int      `json:"fluency_score"`
	ProblemWords  []string `json:"problem_words"`
	FeedbackLevel string   `json:"feedback_level"`
}

// ScoreData 评分数据（从讯飞API返回）
type ScoreData struct {
	OverallScore   int      `json:"overall_score"`
	AccuracyScore  int      `json:"accuracy_score"`
	FluencyScore   int      `json:"fluency_score"`
	IntegrityScore int      `json:"integrity_score"`
	ProblemWords   []string `json:"problem_words"`
	FeedbackLevel  string   `json:"feedback_level"`
	TargetText     string   `json:"target_text"`
}

// Pipeline 任务管道
// 用于处理任务之间的依赖关系
type Pipeline struct {
	service *AsyncService
	evalID  string
	results chan *TaskResult
}

// NewPipeline 创建任务管道
func NewPipeline(service *AsyncService, evalID string) *Pipeline {
	return &Pipeline{
		service: service,
		evalID:  evalID,
		results: make(chan *TaskResult, 10),
	}
}

// ExecuteFeedbackPipeline 执行反馈生成管道
// 1. 生成反馈文本
// 2. 生成反馈音频（依赖文本）
func (p *Pipeline) ExecuteFeedbackPipeline(ctx context.Context, params *EvaluationAsyncParams) error {
	// 第一阶段：提交反馈文本任务
	if err := p.service.SubmitFeedbackTextTask(
		params.EvaluationID,
		params.OverallScore,
		getFirstProblemWord(params.ProblemWords),
		GetFeedbackLevel(params.OverallScore),
		params.TargetText,
	); err != nil {
		return fmt.Errorf("failed to submit feedback text task: %w", err)
	}

	// 注：反馈音频任务需要在反馈文本完成后提交
	// 这通常在 WorkerPool 的 onSuccess 回调中处理
	// 或者使用更复杂的任务依赖管理机制

	return nil
}

// getFirstProblemWord 获取第一个问题单词
func getFirstProblemWord(words []string) string {
	if len(words) > 0 {
		return words[0]
	}
	return ""
}

// Stats 获取异步服务统计信息
func (s *AsyncService) Stats() *WorkerPoolStats {
	return s.pool.Stats()
}

// RegisterHandler 注册任务处理器
func (s *AsyncService) RegisterHandler(taskType TaskType, handler TaskHandler) {
	s.pool.RegisterHandler(taskType, handler)
}

// SetOnSuccess 设置成功回调
func (s *AsyncService) SetOnSuccess(fn func(*TaskResult)) {
	s.pool.SetOnSuccess(fn)
}

// SetOnFailure 设置失败回调
func (s *AsyncService) SetOnFailure(fn func(*TaskResult)) {
	s.pool.SetOnFailure(fn)
}
