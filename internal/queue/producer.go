// Package queue 提供异步任务队列
// 基于 Redis 实现简单的任务队列
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pronunciation-correction-system/internal/cache/redis"
)

// Producer 任务生产者
type Producer struct {
	commands *redis.Commands
}

// NewProducer 创建任务生产者
func NewProducer(commands *redis.Commands) *Producer {
	return &Producer{
		commands: commands,
	}
}

// Publish 发布任务
func (p *Producer) Publish(ctx context.Context, task *Task) error {
	// 设置任务 ID 和创建时间
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	task.Status = TaskStatusPending

	// 序列化任务
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 添加到队列
	queueKey := fmt.Sprintf("queue:%s", task.Type)
	return p.commands.LPush(ctx, queueKey, string(data))
}

// PublishFeedbackGeneration 发布反馈生成任务
func (p *Producer) PublishFeedbackGeneration(ctx context.Context, evaluationID string) error {
	task := &Task{
		Type: TaskTypeFeedbackGeneration,
		Payload: map[string]interface{}{
			"evaluation_id": evaluationID,
		},
	}
	return p.Publish(ctx, task)
}

// PublishAudioGeneration 发布音频生成任务
func (p *Producer) PublishAudioGeneration(ctx context.Context, textID, text string) error {
	task := &Task{
		Type: TaskTypeAudioGeneration,
		Payload: map[string]interface{}{
			"text_id": textID,
			"text":    text,
		},
	}
	return p.Publish(ctx, task)
}

// PublishAudioUpload 发布音频上传任务
func (p *Producer) PublishAudioUpload(ctx context.Context, audioData []byte, filename string) error {
	task := &Task{
		Type: TaskTypeAudioUpload,
		Payload: map[string]interface{}{
			"audio_data": audioData,
			"filename":   filename,
		},
	}
	return p.Publish(ctx, task)
}

// PublishNotification 发布通知任务
func (p *Producer) PublishNotification(ctx context.Context, userID, notificationType, content string) error {
	task := &Task{
		Type: TaskTypeNotification,
		Payload: map[string]interface{}{
			"user_id":           userID,
			"notification_type": notificationType,
			"content":           content,
		},
	}
	return p.Publish(ctx, task)
}

// Task 任务结构
type Task struct {
	ID        string                 `json:"id"`
	Type      TaskType               `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Status    TaskStatus             `json:"status"`
	Retry     int                    `json:"retry"`
	MaxRetry  int                    `json:"max_retry"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeFeedbackGeneration TaskType = "feedback_generation"
	TaskTypeAudioGeneration    TaskType = "audio_generation"
	TaskTypeAudioUpload        TaskType = "audio_upload"
	TaskTypeNotification       TaskType = "notification"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)
