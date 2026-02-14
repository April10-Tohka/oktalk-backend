// Package queue 定义队列相关错误
package queue

import (
	"errors"
)

// 队列错误
var (
	ErrInvalidPayload = errors.New("invalid task payload")
	ErrTaskNotFound   = errors.New("task not found")
	ErrQueueFull      = errors.New("queue is full")
	ErrTaskTimeout    = errors.New("task execution timeout")
	ErrMaxRetryExceeded = errors.New("max retry exceeded")
)
