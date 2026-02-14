// Package async 定义异步任务相关错误
package async

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// 定义错误类型
var (
	// 任务相关错误
	ErrTaskNotFound     = errors.New("task not found")
	ErrTaskTimeout      = errors.New("task execution timeout")
	ErrTaskCancelled    = errors.New("task was cancelled")
	ErrMaxRetryExceeded = errors.New("max retry exceeded")
	ErrNoHandler        = errors.New("no handler registered for task type")
	ErrQueueFull        = errors.New("task queue is full")

	// 处理器相关错误
	ErrInvalidPayload  = errors.New("invalid task payload")
	ErrMissingData     = errors.New("missing required task data")
	ErrProcessingError = errors.New("error processing task")

	// 外部服务相关错误
	ErrLLMUnavailable = errors.New("LLM service unavailable")
	ErrTTSUnavailable = errors.New("TTS service unavailable")
	ErrOSSUnavailable = errors.New("OSS service unavailable")
)

// TaskError 任务错误
type TaskError struct {
	TaskID    string
	TaskType  TaskType
	Err       error
	Retriable bool
}

// Error 实现 error 接口
func (e *TaskError) Error() string {
	return fmt.Sprintf("task error [%s/%s]: %v (retriable: %v)",
		e.TaskID, e.TaskType, e.Err, e.Retriable)
}

// Unwrap 实现 errors.Unwrap
func (e *TaskError) Unwrap() error {
	return e.Err
}

// NewTaskError 创建任务错误
func NewTaskError(taskID string, taskType TaskType, err error, retriable bool) *TaskError {
	return &TaskError{
		TaskID:    taskID,
		TaskType:  taskType,
		Err:       err,
		Retriable: retriable,
	}
}

// IsRetriableError 判断错误是否可重试
func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 TaskError
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		return taskErr.Retriable
	}

	// 网络错误可重试
	if isNetworkError(err) {
		return true
	}

	// 服务端错误可重试
	if isServerError(err) {
		return true
	}

	// 客户端错误不可重试
	if isClientError(err) {
		return false
	}

	// 特定错误类型
	switch {
	case errors.Is(err, ErrLLMUnavailable):
		return true
	case errors.Is(err, ErrTTSUnavailable):
		return true
	case errors.Is(err, ErrOSSUnavailable):
		return true
	case errors.Is(err, ErrTaskTimeout):
		return true
	case errors.Is(err, ErrMaxRetryExceeded):
		return false
	case errors.Is(err, ErrTaskCancelled):
		return false
	case errors.Is(err, ErrInvalidPayload):
		return false
	case errors.Is(err, ErrNoHandler):
		return false
	}

	return false
}

// isNetworkError 检查是否是网络错误
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// 检查网络超时
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// 检查连接错误
	errStr := err.Error()
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"i/o timeout",
		"network is unreachable",
		"broken pipe",
		"EOF",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// isServerError 检查是否是服务端错误（5xx）
func isServerError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	serverKeywords := []string{
		"500",
		"502",
		"503",
		"504",
		"internal server error",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
	}

	for _, keyword := range serverKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// isClientError 检查是否是客户端错误（4xx）
func isClientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	clientKeywords := []string{
		"400",
		"401",
		"403",
		"404",
		"422",
		"bad request",
		"unauthorized",
		"forbidden",
		"not found",
		"validation failed",
	}

	for _, keyword := range clientKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// WrapError 包装错误
func WrapError(taskID string, taskType TaskType, err error) error {
	return NewTaskError(taskID, taskType, err, IsRetriableError(err))
}
