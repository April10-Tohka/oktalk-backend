package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/pkg/logger"
)

// ===================== TaskProcessor 接口 =====================

// TaskProcessor 业务逻辑处理器（由 service 层实现）
type TaskProcessor interface {
	// Process 执行任务
	// 返回值: (result 数据, resultKey, error)
	Process(ctx context.Context, task *Task) (interface{}, string, error)
}

// ResultPersister 结果持久化接口（由 service 层实现，Worker 不直接引用 DB）
type ResultPersister interface {
	// SaveResult 将结果写入数据库
	SaveResult(ctx context.Context, task *Task, result interface{}) error
}

// ===================== WorkerPool =====================

// WorkerPool 工作池
type WorkerPool struct {
	name        string
	taskChannel chan *Task
	workerCount int
	processor   TaskProcessor
	persister   ResultPersister
	taskCache   *cache.TaskCache
	chatCache   *cache.ChatCache
	evalCache   *cache.EvalCache
	reportCache *cache.ReportCache
	logger      *slog.Logger
	wg          sync.WaitGroup
	ctx         context.Context // 用于取消所有 worker goroutine
	cancel      context.CancelFunc
}

// NewWorkerPool 创建工作池
func NewWorkerPool(
	name string,
	workerCount int,
	bufferSize int,
	processor TaskProcessor,
	persister ResultPersister,
	taskCache *cache.TaskCache,
	chatCache *cache.ChatCache,
	evalCache *cache.EvalCache,
	reportCache *cache.ReportCache,
	logger *slog.Logger,
) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		name:        name,
		taskChannel: make(chan *Task, bufferSize),
		workerCount: workerCount,
		processor:   processor,
		persister:   persister,
		taskCache:   taskCache,
		chatCache:   chatCache,
		evalCache:   evalCache,
		reportCache: reportCache,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动所有 worker goroutine
func (p *WorkerPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	logger.Info("WorkerPool started",
		slog.String("name", p.name),
		slog.Int("workers", p.workerCount),
	)
}

// Stop 优雅关闭，drainTimeout 默认 30s
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.taskChannel)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("WorkerPool stopped gracefully", slog.String("name", p.name))
	case <-time.After(30 * time.Second):
		logger.Warn("WorkerPool stop timeout, force shutdown", slog.String("name", p.name))
	}
}

// Submit 提交任务到工作池
func (p *WorkerPool) Submit(task *Task) error {
	select {
	case p.taskChannel <- task:
		return nil
	default:
		return fmt.Errorf("worker pool %s: task channel full", p.name)
	}
}

// worker 单个 worker goroutine 的处理流程
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for task := range p.taskChannel {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		p.processTask(task)
	}
}

// processTask 处理单个任务（完整流程）
func (p *WorkerPool) processTask(task *Task) {
	taskCtx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	// Step 1: 更新状态为 processing
	if err := p.taskCache.UpdateTaskStatus(taskCtx, task.ID, "processing"); err != nil {
		logger.Error("update task status to processing failed",
			slog.String("task_id", task.ID), slog.String("error", err.Error()))
	}

	// Step 2: 执行业务逻辑
	result, resultKey, err := p.processor.Process(taskCtx, task)

	if err != nil {
		// Step 3a: 失败
		if updateErr := p.taskCache.UpdateTaskStatus(taskCtx, task.ID, "failed", err.Error()); updateErr != nil {
			logger.Error("update task status to failed err",
				slog.String("task_id", task.ID), slog.String("error", updateErr.Error()))
		}
		logger.Error("task processing failed",
			slog.String("pool", p.name),
			slog.String("task_id", task.ID),
			slog.String("error", err.Error()),
		)
	} else {
		// Step 3b: 成功
		// Step 3b-1: 写结果缓存
		p.setResultByType(taskCtx, task.Type, resultKey, result)

		// Step 3b-2: 更新任务状态
		if updateErr := p.taskCache.UpdateTaskStatus(taskCtx, task.ID, "success"); updateErr != nil {
			logger.Error("update task status to success err",
				slog.String("task_id", task.ID), slog.String("error", updateErr.Error()))
		}
		if setErr := p.taskCache.SetResultKey(taskCtx, task.ID, resultKey); setErr != nil {
			logger.Error("set task result key err",
				slog.String("task_id", task.ID), slog.String("error", setErr.Error()))
		}

		// Step 3b-3: 异步持久化到 DB（独立 goroutine + 10s 超时）
		if p.persister != nil {
			go func() {
				dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer dbCancel()
				if dbErr := p.persister.SaveResult(dbCtx, task, result); dbErr != nil {
					logger.Error("DEAD_LETTER: db persist failed",
						slog.String("task_id", task.ID),
						slog.String("type", task.Type),
						slog.String("error", dbErr.Error()),
					)
				}
			}()
		}

		// Step 3b-4: 失效列表缓存
		p.invalidateHistory(taskCtx, task.Type, task.UserID)
	}

	// Step 4: 从 pending ZSet 移除
	if removeErr := p.taskCache.RemovePendingTask(taskCtx, task.Type, task.ID); removeErr != nil {
		logger.Error("remove pending task failed",
			slog.String("task_id", task.ID), slog.String("error", removeErr.Error()))
	}
}

// setResultByType 根据任务类型写入对应结果缓存
func (p *WorkerPool) setResultByType(ctx context.Context, taskType, resultKey string, result interface{}) {
	var err error
	switch taskType {
	case "chat":
		if cr, ok := result.(*cache.ChatResult); ok {
			err = p.chatCache.SetChatResult(ctx, resultKey, cr)
		}
	case "evaluate":
		if er, ok := result.(*cache.EvalResult); ok {
			err = p.evalCache.SetEvalResult(ctx, resultKey, er)
		}
	case "report":
		err = p.reportCache.SetReportResult(ctx, resultKey, result)
	}
	if err != nil {
		logger.Error("set result cache failed",
			slog.String("type", taskType),
			slog.String("result_key", resultKey),
			slog.String("error", err.Error()),
		)
	}
}

// invalidateHistory 根据任务类型删除对应历史列表缓存
func (p *WorkerPool) invalidateHistory(ctx context.Context, taskType, userID string) {
	var err error
	switch taskType {
	case "chat":
		err = p.chatCache.InvalidateChatHistory(ctx, userID)
	case "evaluate":
		err = p.evalCache.InvalidateEvalHistory(ctx, userID)
	case "report":
		err = p.reportCache.InvalidateReportHistory(ctx, userID)
	}
	if err != nil {
		logger.Warn("invalidate history cache failed",
			slog.String("type", taskType),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}
}
