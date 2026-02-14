// Package async 提供异步任务定义
// 基于 Channel + Goroutine + Worker Pool 架构
package async

import (
	"time"
)

// TaskType 定义任务类型
type TaskType string

const (
	// 反馈生成任务
	TaskGenerateFeedbackText  TaskType = "generate_feedback_text"
	TaskGenerateFeedbackAudio TaskType = "generate_feedback_audio"

	// 示范音频生成任务
	TaskGenerateDemoAudio TaskType = "generate_demo_audio"

	// 其他任务
	TaskUploadAudio    TaskType = "upload_audio"
	TaskNotification   TaskType = "notification"
	TaskUpdateDatabase TaskType = "update_database"
	TaskUpdateCache    TaskType = "update_cache"
)

// EvaluationTask 评测异步任务
type EvaluationTask struct {
	ID           string                 `json:"id"`            // evaluation_id
	Type         TaskType               `json:"type"`          // 任务类型
	Priority     int                    `json:"priority"`      // 优先级 (1=最高)
	Data         map[string]interface{} `json:"data"`          // 任务数据
	RetryCount   int                    `json:"retry_count"`   // 当前重试次数
	MaxRetries   int                    `json:"max_retries"`   // 最大重试次数
	CreatedAt    time.Time              `json:"created_at"`    // 创建时间
	ExecuteAfter time.Time              `json:"execute_after"` // 延迟执行时间(用于重试)
}

// NewEvaluationTask 创建评测任务
func NewEvaluationTask(id string, taskType TaskType, data map[string]interface{}) *EvaluationTask {
	return &EvaluationTask{
		ID:         id,
		Type:       taskType,
		Priority:   DefaultPriority,
		Data:       data,
		MaxRetries: DefaultMaxRetries,
		CreatedAt:  time.Now(),
	}
}

// WithPriority 设置任务优先级
func (t *EvaluationTask) WithPriority(priority int) *EvaluationTask {
	t.Priority = priority
	return t
}

// WithMaxRetries 设置最大重试次数
func (t *EvaluationTask) WithMaxRetries(maxRetries int) *EvaluationTask {
	t.MaxRetries = maxRetries
	return t
}

// WithExecuteAfter 设置延迟执行时间
func (t *EvaluationTask) WithExecuteAfter(executeAfter time.Time) *EvaluationTask {
	t.ExecuteAfter = executeAfter
	return t
}

// CanRetry 检查是否可以重试
func (t *EvaluationTask) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// IncrementRetry 增加重试次数
func (t *EvaluationTask) IncrementRetry() {
	t.RetryCount++
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	TaskType  TaskType               `json:"task_type"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	CreatedAt time.Time              `json:"created_at"`
}

// NewTaskResult 创建任务结果
func NewTaskResult(taskID string, taskType TaskType) *TaskResult {
	return &TaskResult{
		TaskID:    taskID,
		TaskType:  taskType,
		CreatedAt: time.Now(),
	}
}

// SetSuccess 设置成功结果
func (r *TaskResult) SetSuccess(data map[string]interface{}) *TaskResult {
	r.Success = true
	r.Data = data
	return r
}

// SetError 设置错误结果
func (r *TaskResult) SetError(err error) *TaskResult {
	r.Success = false
	r.Error = err
	return r
}

// SetDuration 设置执行时长
func (r *TaskResult) SetDuration(duration time.Duration) *TaskResult {
	r.Duration = duration
	return r
}

// 常量定义
const (
	DefaultPriority   = 5  // 默认优先级
	HighPriority      = 1  // 高优先级
	LowPriority       = 10 // 低优先级
	DefaultMaxRetries = 3  // 默认最大重试次数
)

// 任务数据 Key 常量
const (
	// 通用
	DataKeyEvaluationID = "evaluation_id"
	DataKeyUserID       = "user_id"
	DataKeyTargetText   = "target_text"

	// 评分相关
	DataKeyScore        = "score"
	DataKeyOverallScore = "overall_score"
	DataKeyProblemWord  = "problem_word"
	DataKeyProblemWords = "problem_words"
	DataKeyLevel        = "level"
	DataKeyFeedbackLevel = "feedback_level"

	// 示范音频相关
	DataKeyDemoText = "demo_text"
	DataKeyDemoType = "demo_type" // "word" or "sentence"

	// 反馈相关
	DataKeyFeedbackText     = "feedback_text"
	DataKeyFeedbackAudioURL = "feedback_audio_url"
	DataKeyDemoAudioURL     = "demo_audio_url"

	// 音频相关
	DataKeyAudioData = "audio_data"
	DataKeyAudioURL  = "audio_url"
	DataKeyFilename  = "filename"
	DataKeyDuration  = "duration"

	// 通知相关
	DataKeyNotificationType = "notification_type"
	DataKeyContent          = "content"
)

// DemoType 示范音频类型
const (
	DemoTypeWord     = "word"
	DemoTypeSentence = "sentence"
)

// FeedbackLevel 反馈级别
const (
	FeedbackLevelS = "S" // 90分以上
	FeedbackLevelA = "A" // 70-89分
	FeedbackLevelB = "B" // 50-69分
	FeedbackLevelC = "C" // 50分以下
)

// GetFeedbackLevel 根据分数获取反馈级别
func GetFeedbackLevel(score int) string {
	if score >= 90 {
		return FeedbackLevelS
	} else if score >= 70 {
		return FeedbackLevelA
	} else if score >= 50 {
		return FeedbackLevelB
	}
	return FeedbackLevelC
}

// GetString 从任务数据中获取字符串
func (t *EvaluationTask) GetString(key string) string {
	if v, ok := t.Data[key].(string); ok {
		return v
	}
	return ""
}

// GetInt 从任务数据中获取整数
func (t *EvaluationTask) GetInt(key string) int {
	switch v := t.Data[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// GetStringSlice 从任务数据中获取字符串切片
func (t *EvaluationTask) GetStringSlice(key string) []string {
	if v, ok := t.Data[key].([]string); ok {
		return v
	}
	if v, ok := t.Data[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
