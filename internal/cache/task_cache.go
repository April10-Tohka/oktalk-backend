package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ===================== TaskMeta 结构 =====================

// TaskMeta 任务元数据
type TaskMeta struct {
	TaskID       string `json:"task_id"`
	Type         string `json:"type"`          // chat / evaluate / report
	Status       string `json:"status"`        // pending / processing / success / failed
	CurrentStage string `json:"current_stage"` // queued / asr / llm / tts / oss / db / completed
	ResultKey    string `json:"result_key"`
	Error        string `json:"error,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	UserID       string `json:"user_id"`
}

// ===================== TaskCache =====================

// TaskCache 任务缓存操作
type TaskCache struct {
	rdb *redis.Client
}

// NewTaskCache 创建 TaskCache
func NewTaskCache(rdb *redis.Client) *TaskCache {
	return &TaskCache{rdb: rdb}
}

// SetTaskMeta 写入任务元数据
func (c *TaskCache) SetTaskMeta(ctx context.Context, taskID string, meta *TaskMeta) error {
	key := fmt.Sprintf(KeyTaskMeta, taskID)
	return SetJSON(ctx, c.rdb, key, meta, TTLTaskMeta)
}

// GetTaskMeta 获取任务元数据
func (c *TaskCache) GetTaskMeta(ctx context.Context, taskID string) (*TaskMeta, error) {
	key := fmt.Sprintf(KeyTaskMeta, taskID)
	meta, found, err := GetJSON[TaskMeta](ctx, c.rdb, key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return meta, nil
}

// UpdateTaskStatus 只更新 status、updated_at、error 字段
func (c *TaskCache) UpdateTaskStatus(ctx context.Context, taskID, status string, errMsg ...string) error {
	meta, err := c.GetTaskMeta(ctx, taskID)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("task meta not found: %s", taskID)
	}

	meta.Status = status
	meta.UpdatedAt = time.Now().Unix()
	if len(errMsg) > 0 {
		meta.Error = errMsg[0]
	}

	return c.SetTaskMeta(ctx, taskID, meta)
}

// UpdateTaskStage 更新任务当前阶段
func (c *TaskCache) UpdateTaskStage(ctx context.Context, taskID, stage string) error {
	meta, err := c.GetTaskMeta(ctx, taskID)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("task meta not found: %s", taskID)
	}

	meta.CurrentStage = stage
	meta.UpdatedAt = time.Now().Unix()
	return c.SetTaskMeta(ctx, taskID, meta)
}

// SetResultKey 更新 TaskMeta 的 ResultKey
func (c *TaskCache) SetResultKey(ctx context.Context, taskID, resultKey string) error {
	meta, err := c.GetTaskMeta(ctx, taskID)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("task meta not found: %s", taskID)
	}

	meta.ResultKey = resultKey
	meta.UpdatedAt = time.Now().Unix()
	return c.SetTaskMeta(ctx, taskID, meta)
}

// ===================== Pending ZSet =====================

// AddPendingTask 添加到 pending ZSet（score = createdAt timestamp）
func (c *TaskCache) AddPendingTask(ctx context.Context, taskType, taskID string, createdAt int64) error {
	key := fmt.Sprintf(KeyTaskPending, taskType)
	return ZAdd(ctx, c.rdb, key, taskID, float64(createdAt))
}

// RemovePendingTask 从 pending ZSet 移除
func (c *TaskCache) RemovePendingTask(ctx context.Context, taskType, taskID string) error {
	key := fmt.Sprintf(KeyTaskPending, taskType)
	return ZRem(ctx, c.rdb, key, taskID)
}

// ScanPendingTasks 扫描 pending ZSet（分批）
func (c *TaskCache) ScanPendingTasks(ctx context.Context, taskType string, cursor uint64, count int64) ([]string, uint64, error) {
	key := fmt.Sprintf(KeyTaskPending, taskType)
	return ZScan(ctx, c.rdb, key, cursor, count)
}
