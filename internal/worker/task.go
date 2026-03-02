// Package worker 提供异步任务处理系统
package worker

import "time"

// ===================== Task 数据结构 =====================

// Task 异步任务
type Task struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`      // chat / evaluate / report
	DomainID   string    `json:"domain_id"` // 业务主键ID，用于映射如 chat的UserMessageID 或 evaluate的EvaluationID
	UserID     string    `json:"user_id"`
	Payload    []byte    `json:"payload"` // JSON 序列化的业务参数
	CreatedAt  time.Time `json:"created_at"`
	RetryCount int       `json:"retry_count"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string      `json:"task_id"`
	ResultKey string      `json:"result_key"`
	Data      interface{} `json:"data"`
}

// TaskQueryResult 任务查询结果（返回给前端）
type TaskQueryResult struct {
	TaskID    string      `json:"task_id"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	CreatedAt int64       `json:"created_at"`
	UpdatedAt int64       `json:"updated_at"`
}
