// Package async 提供 Worker Pool 实现
// 基于 Channel + Goroutine 的异步任务处理池
package async

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskHandler 任务处理器接口
type TaskHandler interface {
	// Handle 处理任务
	Handle(ctx context.Context, task *EvaluationTask) (*TaskResult, error)
}

// WorkerPool 异步任务工作池
type WorkerPool struct {
	workerCount int
	taskQueue   chan *EvaluationTask
	resultQueue chan *TaskResult
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	handlers    map[TaskType]TaskHandler
	mu          sync.RWMutex

	// 回调函数
	onSuccess func(*TaskResult)
	onFailure func(*TaskResult)
}

// WorkerPoolConfig Worker Pool 配置
type WorkerPoolConfig struct {
	WorkerCount     int           // Worker 数量
	QueueSize       int           // 任务队列大小
	ShutdownTimeout time.Duration // 优雅关闭超时
}

// DefaultWorkerPoolConfig 默认配置
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		WorkerCount:     10,
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
	}
}

// NewWorkerPool 创建工作池
func NewWorkerPool(cfg *WorkerPoolConfig) *WorkerPool {
	if cfg == nil {
		cfg = DefaultWorkerPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workerCount: cfg.WorkerCount,
		taskQueue:   make(chan *EvaluationTask, cfg.QueueSize),
		resultQueue: make(chan *TaskResult, cfg.QueueSize),
		wg:          &sync.WaitGroup{},
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[TaskType]TaskHandler),
	}
}

// RegisterHandler 注册任务处理器
func (p *WorkerPool) RegisterHandler(taskType TaskType, handler TaskHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[taskType] = handler
	log.Printf("[WorkerPool] Registered handler for task type: %s", taskType)
}

// SetOnSuccess 设置成功回调
func (p *WorkerPool) SetOnSuccess(fn func(*TaskResult)) {
	p.onSuccess = fn
}

// SetOnFailure 设置失败回调
func (p *WorkerPool) SetOnFailure(fn func(*TaskResult)) {
	p.onFailure = fn
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	// 启动 worker goroutines
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// 启动结果处理器
	go p.resultProcessor()

	log.Printf("[WorkerPool] Started with %d workers", p.workerCount)
}

// worker 工作协程
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("[Worker-%d] Shutting down", id)
			return

		case task := <-p.taskQueue:
			if task == nil {
				continue
			}

			// 检查是否需要延迟执行
			if !task.ExecuteAfter.IsZero() && time.Now().Before(task.ExecuteAfter) {
				delay := time.Until(task.ExecuteAfter)
				if delay > 0 {
					log.Printf("[Worker-%d] Task %s delayed for %v", id, task.ID, delay)
					select {
					case <-time.After(delay):
					case <-p.ctx.Done():
						return
					}
				}
			}

			// 执行任务
			result := p.executeTask(id, task)

			// 发送结果
			select {
			case p.resultQueue <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// executeTask 执行任务
func (p *WorkerPool) executeTask(workerID int, task *EvaluationTask) *TaskResult {
	startTime := time.Now()

	p.mu.RLock()
	handler, exists := p.handlers[task.Type]
	p.mu.RUnlock()

	if !exists {
		log.Printf("[Worker-%d] No handler for task type: %s", workerID, task.Type)
		return NewTaskResult(task.ID, task.Type).
			SetError(fmt.Errorf("no handler for task type: %s", task.Type)).
			SetDuration(time.Since(startTime))
	}

	log.Printf("[Worker-%d] Processing task %s (type: %s, retry: %d/%d)",
		workerID, task.ID, task.Type, task.RetryCount, task.MaxRetries)

	// 执行任务
	result, err := handler.Handle(p.ctx, task)
	if err != nil {
		log.Printf("[Worker-%d] Task %s failed: %v (retry: %d/%d)",
			workerID, task.ID, err, task.RetryCount, task.MaxRetries)

		// 判断是否需要重试
		if task.CanRetry() {
			p.retryTask(task)
		}

		if result == nil {
			result = NewTaskResult(task.ID, task.Type)
		}
		result.SetError(err).SetDuration(time.Since(startTime))
		return result
	}

	if result == nil {
		result = NewTaskResult(task.ID, task.Type).SetSuccess(nil)
	}
	result.SetDuration(time.Since(startTime))

	log.Printf("[Worker-%d] Task %s completed in %v", workerID, task.ID, result.Duration)
	return result
}

// retryTask 重试任务
func (p *WorkerPool) retryTask(task *EvaluationTask) {
	task.IncrementRetry()

	// 指数退避: 2^n 秒
	backoff := time.Duration(1<<uint(task.RetryCount)) * time.Second
	task.ExecuteAfter = time.Now().Add(backoff)

	log.Printf("[WorkerPool] Scheduling retry for task %s in %v (retry: %d/%d)",
		task.ID, backoff, task.RetryCount, task.MaxRetries)

	// 重新入队
	go func() {
		select {
		case p.taskQueue <- task:
		case <-p.ctx.Done():
		}
	}()
}

// resultProcessor 结果处理器
func (p *WorkerPool) resultProcessor() {
	for {
		select {
		case <-p.ctx.Done():
			return

		case result := <-p.resultQueue:
			if result == nil {
				continue
			}

			if result.Success {
				p.handleSuccess(result)
			} else {
				p.handleFailure(result)
			}
		}
	}
}

// handleSuccess 处理成功结果
func (p *WorkerPool) handleSuccess(result *TaskResult) {
	log.Printf("[WorkerPool] Task %s succeeded (type: %s, duration: %v)",
		result.TaskID, result.TaskType, result.Duration)

	if p.onSuccess != nil {
		p.onSuccess(result)
	}
}

// handleFailure 处理失败结果
func (p *WorkerPool) handleFailure(result *TaskResult) {
	log.Printf("[WorkerPool] Task %s failed (type: %s, error: %v)",
		result.TaskID, result.TaskType, result.Error)

	if p.onFailure != nil {
		p.onFailure(result)
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(task *EvaluationTask) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	select {
	case p.taskQueue <- task:
		log.Printf("[WorkerPool] Task %s submitted (type: %s, priority: %d)",
			task.ID, task.Type, task.Priority)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue full, submit timeout")
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// SubmitWithTimeout 提交任务（带超时）
func (p *WorkerPool) SubmitWithTimeout(task *EvaluationTask, timeout time.Duration) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	select {
	case p.taskQueue <- task:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("task queue full")
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// Shutdown 优雅关闭
func (p *WorkerPool) Shutdown(timeout time.Duration) {
	log.Println("[WorkerPool] Shutting down...")

	p.cancel()

	// 等待所有 worker 完成
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[WorkerPool] Shutdown completed")
	case <-time.After(timeout):
		log.Println("[WorkerPool] Shutdown timeout, forcing exit")
	}
}

// Stats 获取工作池统计信息
func (p *WorkerPool) Stats() *WorkerPoolStats {
	return &WorkerPoolStats{
		WorkerCount:     p.workerCount,
		QueueSize:       cap(p.taskQueue),
		PendingTasks:    len(p.taskQueue),
		PendingResults:  len(p.resultQueue),
		RegisteredTypes: p.registeredTypes(),
	}
}

// registeredTypes 获取已注册的任务类型
func (p *WorkerPool) registeredTypes() []TaskType {
	p.mu.RLock()
	defer p.mu.RUnlock()

	types := make([]TaskType, 0, len(p.handlers))
	for t := range p.handlers {
		types = append(types, t)
	}
	return types
}

// WorkerPoolStats 工作池统计信息
type WorkerPoolStats struct {
	WorkerCount     int        `json:"worker_count"`
	QueueSize       int        `json:"queue_size"`
	PendingTasks    int        `json:"pending_tasks"`
	PendingResults  int        `json:"pending_results"`
	RegisteredTypes []TaskType `json:"registered_types"`
}
