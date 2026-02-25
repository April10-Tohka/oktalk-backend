package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/pkg/logger"
)

// ===================== Manager =====================

// ManagerConfig Worker Manager 配置
type ManagerConfig struct {
	ChatWorkers   int `json:"chat_workers"`
	ChatBuffer    int `json:"chat_buffer"`
	EvalWorkers   int `json:"eval_workers"`
	EvalBuffer    int `json:"eval_buffer"`
	ReportWorkers int `json:"report_workers"`
	ReportBuffer  int `json:"report_buffer"`
}

// DefaultManagerConfig 默认并发配置
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		ChatWorkers:   10,
		ChatBuffer:    100,
		EvalWorkers:   15,
		EvalBuffer:    150,
		ReportWorkers: 5,
		ReportBuffer:  50,
	}
}

// Manager 任务管理器（入口）
type Manager struct {
	chatPool    *WorkerPool
	evalPool    *WorkerPool
	reportPool  *WorkerPool
	taskCache   *cache.TaskCache
	chatCache   *cache.ChatCache
	evalCache   *cache.EvalCache
	reportCache *cache.ReportCache
	logger      *slog.Logger
}

// ProcessorSet 三种类型的处理器
type ProcessorSet struct {
	Chat   TaskProcessor
	Eval   TaskProcessor
	Report TaskProcessor
}

// PersisterSet 三种类型的持久化器
type PersisterSet struct {
	Chat   ResultPersister
	Eval   ResultPersister
	Report ResultPersister
}

// NewManager 创建 Manager
func NewManager(
	cfg ManagerConfig,
	processors ProcessorSet,
	persisters PersisterSet,
	taskCache *cache.TaskCache,
	chatCache *cache.ChatCache,
	evalCache *cache.EvalCache,
	reportCache *cache.ReportCache,
	logger *slog.Logger,
) *Manager {
	m := &Manager{
		taskCache:   taskCache,
		chatCache:   chatCache,
		evalCache:   evalCache,
		reportCache: reportCache,
		logger:      logger,
	}

	m.chatPool = NewWorkerPool("chat", cfg.ChatWorkers, cfg.ChatBuffer,
		processors.Chat, persisters.Chat, taskCache, chatCache, evalCache, reportCache, logger)
	m.evalPool = NewWorkerPool("eval", cfg.EvalWorkers, cfg.EvalBuffer,
		processors.Eval, persisters.Eval, taskCache, chatCache, evalCache, reportCache, logger)
	m.reportPool = NewWorkerPool("report", cfg.ReportWorkers, cfg.ReportBuffer,
		processors.Report, persisters.Report, taskCache, chatCache, evalCache, reportCache, logger)

	return m
}

// Start 启动所有 WorkerPool
func (m *Manager) Start() {
	m.chatPool.Start()
	m.evalPool.Start()
	m.reportPool.Start()
	m.logger.Info("Worker Manager started")
}

// Stop 优雅关闭所有 WorkerPool
func (m *Manager) Stop() {
	m.chatPool.Stop()
	m.evalPool.Stop()
	m.reportPool.Stop()
	m.logger.Info("Worker Manager stopped")
}

// SubmitTask 提交任务
func (m *Manager) SubmitTask(ctx context.Context, task *Task) (string, error) {
	// 1. 生成 task_id
	task.ID = fmt.Sprintf("%s_%d", task.Type, time.Now().UnixNano())
	task.CreatedAt = time.Now()

	// 2. 构建 TaskMeta
	var resultKey string
	switch task.Type {
	case "chat":
		resultKey = fmt.Sprintf(cache.KeyChatResult, task.ID)
	case "evaluate":
		resultKey = fmt.Sprintf(cache.KeyEvalResult, task.ID)
	case "report":
		resultKey = fmt.Sprintf(cache.KeyReportResult, task.ID)
	default:
		return "", fmt.Errorf("unknown task type: %s", task.Type)
	}

	meta := &cache.TaskMeta{
		TaskID:    task.ID,
		Type:      task.Type,
		Status:    "pending",
		ResultKey: resultKey,
		CreatedAt: task.CreatedAt.Unix(),
		UpdatedAt: task.CreatedAt.Unix(),
		UserID:    task.UserID,
	}

	// 3. 写入 TaskMeta
	if err := m.taskCache.SetTaskMeta(ctx, task.ID, meta); err != nil {
		return "", fmt.Errorf("set task meta: %w", err)
	}
	// TODO: 写入 DB

	// 4. 加入 Pending ZSet
	if err := m.taskCache.AddPendingTask(ctx, task.Type, task.ID, task.CreatedAt.Unix()); err != nil {
		logger.Warn("add pending task failed", "task_id", task.ID, "error", err)
	}

	// 5. 路由到对应 Pool
	pool := m.getPool(task.Type)
	if pool == nil {
		return "", fmt.Errorf("no pool for task type: %s", task.Type)
	}
	if err := pool.Submit(task); err != nil {
		return "", fmt.Errorf("submit to pool: %w", err)
	}

	logger.Info("task submitted",
		slog.String("task_id", task.ID),
		slog.String("type", task.Type),
		slog.String("user_id", task.UserID),
	)
	return task.ID, nil
}

// QueryTask 查询任务状态
func (m *Manager) QueryTask(ctx context.Context, taskID string) (*TaskQueryResult, error) {
	meta, err := m.taskCache.GetTaskMeta(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	result := &TaskQueryResult{
		TaskID:    meta.TaskID,
		Type:      meta.Type,
		Status:    meta.Status,
		Error:     meta.Error,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
	}

	// 若 status == success，读取完整结果
	if meta.Status == "success" && meta.ResultKey != "" {
		data, err := m.getResultByType(ctx, meta.Type, meta.ResultKey)
		if err == nil {
			result.Data = data
		}
	}

	return result, nil
}

// getPool 根据任务类型获取对应 Pool
func (m *Manager) getPool(taskType string) *WorkerPool {
	switch taskType {
	case "chat":
		return m.chatPool
	case "evaluate":
		return m.evalPool
	case "report":
		return m.reportPool
	default:
		return nil
	}
}

// getResultByType 根据类型从缓存读取结果
func (m *Manager) getResultByType(ctx context.Context, taskType, resultKey string) (interface{}, error) {
	switch taskType {
	case "chat":
		// resultKey 格式: chat:result:{task_id}，提取 task_id
		var taskID string
		fmt.Sscanf(resultKey, "chat:result:%s", &taskID)
		r, found, err := m.chatCache.GetChatResult(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		return r, nil
	case "evaluate":
		var evalID string
		fmt.Sscanf(resultKey, "evaluate:result:%s", &evalID)
		r, found, err := m.evalCache.GetEvalResult(ctx, evalID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		return r, nil
	case "report":
		var reportID string
		fmt.Sscanf(resultKey, "report:result:%s", &reportID)
		r, found, err := m.reportCache.GetReportResult(ctx, reportID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		return r, nil
	default:
		return nil, nil
	}
}

// LoadTaskPayload 从 Payload 解码到目标结构
func LoadTaskPayload[T any](task *Task) (*T, error) {
	var payload T
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal task payload: %w", err)
	}
	return &payload, nil
}
