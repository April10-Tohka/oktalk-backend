// Package queue 提供任务消费者
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// Consumer 任务消费者
type Consumer struct {
	commands  *redis.Commands
	processor *Processor
	handlers  map[TaskType]TaskHandler
	stopCh    chan struct{}
}

// TaskHandler 任务处理器
type TaskHandler func(ctx context.Context, task *Task) error

// NewConsumer 创建任务消费者
func NewConsumer(commands *redis.Commands) *Consumer {
	return &Consumer{
		commands: commands,
		handlers: make(map[TaskType]TaskHandler),
		stopCh:   make(chan struct{}),
	}
}

// RegisterHandler 注册任务处理器
func (c *Consumer) RegisterHandler(taskType TaskType, handler TaskHandler) {
	c.handlers[taskType] = handler
}

// Start 启动消费者
func (c *Consumer) Start(ctx context.Context) {
	for taskType := range c.handlers {
		go c.consumeQueue(ctx, taskType)
	}
}

// Stop 停止消费者
func (c *Consumer) Stop() {
	close(c.stopCh)
}

// consumeQueue 消费队列
func (c *Consumer) consumeQueue(ctx context.Context, taskType TaskType) {
	queueKey := fmt.Sprintf("queue:%s", taskType)
	
	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// 从队列中获取任务
			data, err := c.commands.RPop(ctx, queueKey)
			if err != nil {
				if !redis.IsNil(err) {
					log.Printf("Error popping from queue %s: %v", queueKey, err)
				}
				// 队列为空，等待一段时间
				time.Sleep(time.Second)
				continue
			}

			// 解析任务
			var task Task
			if err := json.Unmarshal([]byte(data), &task); err != nil {
				log.Printf("Error unmarshaling task: %v", err)
				continue
			}

			// 处理任务
			c.processTask(ctx, &task)
		}
	}
}

// processTask 处理单个任务
func (c *Consumer) processTask(ctx context.Context, task *Task) {
	handler, ok := c.handlers[task.Type]
	if !ok {
		log.Printf("No handler for task type: %s", task.Type)
		return
	}

	// 更新任务状态
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	// 执行任务
	if err := handler(ctx, task); err != nil {
		log.Printf("Task %s failed: %v", task.ID, err)
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		task.Retry++

		// 重试逻辑
		if task.Retry < task.MaxRetry {
			c.retryTask(ctx, task)
		}
	} else {
		task.Status = TaskStatusCompleted
	}

	task.UpdatedAt = time.Now()
}

// retryTask 重试任务
func (c *Consumer) retryTask(ctx context.Context, task *Task) {
	// 延迟重试
	delay := time.Duration(task.Retry) * time.Second
	time.Sleep(delay)

	// 重新入队
	data, _ := json.Marshal(task)
	queueKey := fmt.Sprintf("queue:%s", task.Type)
	c.commands.LPush(ctx, queueKey, string(data))
}
