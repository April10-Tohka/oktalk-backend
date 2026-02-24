package worker

import (
	"context"
	"log/slog"
	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/pkg/logger"
	"strconv"
	"time"
)

// ===================== 任务回补机制 =====================

const (
	recoveryInterval = 30 * time.Second // 扫描间隔
	staleThreshold   = 300              // 超时阈值（秒）
	scanBatchSize    = int64(100)       // 每批扫描数量
)

// StartRecovery 启动定时任务回补
func (m *Manager) StartRecovery(ctx context.Context) {
	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()

	logger.Info("Task recovery started", slog.Duration("interval", recoveryInterval))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Task recovery stopped")
			return
		case <-ticker.C:
			m.recoverTaskType(ctx, "chat")
			m.recoverTaskType(ctx, "evaluate")
			m.recoverTaskType(ctx, "report")
		}
	}
}

// recoverTaskType 扫描指定类型的 pending 任务，回补超时任务
func (m *Manager) recoverTaskType(ctx context.Context, taskType string) {
	var cursor uint64

	for {
		// 使用 ZSCAN 分批遍历
		results, nextCursor, err := m.taskCache.ScanPendingTasks(ctx, taskType, cursor, scanBatchSize)
		if err != nil {
			logger.Warn("scan pending tasks failed",
				slog.String("type", taskType), slog.String("error", err.Error()))
			return
		}

		// ZSCAN 结果是 [member, score, member, score, ...] 交替
		for i := 0; i+1 < len(results); i += 2 {
			taskID := results[i]
			// results[i+1] 是 score（可选解析）

			m.tryRecoverTask(ctx, taskType, taskID)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// tryRecoverTask 尝试回补单个任务
func (m *Manager) tryRecoverTask(ctx context.Context, taskType, taskID string) {
	meta, err := m.taskCache.GetTaskMeta(ctx, taskID)
	if err != nil {
		logger.Warn("get task meta for recovery failed",
			slog.String("task_id", taskID), slog.String("error", err.Error()))
		return
	}

	if meta == nil {
		// meta 不存在（Redis 过期），从 ZSet 移除
		_ = m.taskCache.RemovePendingTask(ctx, taskType, taskID)
		logger.Info("removed expired pending task",
			slog.String("task_id", taskID), slog.String("type", taskType))
		return
	}

	// 只回补 pending 状态且超时的任务
	if meta.Status != "pending" {
		return
	}

	elapsed := time.Now().Unix() - meta.CreatedAt
	if elapsed <= staleThreshold {
		return
	}

	// 构建重试 Task
	task := &Task{
		ID:         taskID,
		Type:       taskType,
		UserID:     meta.UserID,
		Payload:    nil, // 从 DB 重新加载需 service 层配合，暂置 nil
		CreatedAt:  time.Unix(meta.CreatedAt, 0),
		RetryCount: 1,
	}

	// 尝试解析 retry count（从 ResultKey 借用或 meta 自带）
	if meta.Error != "" {
		// 可以从 error 中提取 retry count，暂简化处理
		task.RetryCount++
	}

	pool := m.getPool(taskType)
	if pool == nil {
		return
	}

	if err := pool.Submit(task); err != nil {
		logger.Warn("re-submit recovered task failed",
			slog.String("task_id", taskID), slog.String("error", err.Error()))
		return
	}

	logger.Warn("task recovered",
		slog.String("task_id", taskID),
		slog.String("type", taskType),
		slog.Int64("elapsed_seconds", elapsed),
	)
}

// parseScore 辅助解析 ZSet score（安全解析）
func parseScore(s string) int64 {
	f, _ := strconv.ParseFloat(s, 64)
	return int64(f)
}

// ensure interface marker to suppress 'unused' if needed
var _ = (*cache.TaskMeta)(nil)
