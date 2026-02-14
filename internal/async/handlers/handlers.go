// Package handlers 提供所有任务处理器的统一导出
package handlers

import (
	"pronunciation-correction-system/internal/async"
	"pronunciation-correction-system/internal/cache"
)

// Handlers 处理器集合
type Handlers struct {
	FeedbackText  *FeedbackTextHandler
	FeedbackAudio *FeedbackAudioHandler
	DemoAudio     *DemoAudioHandler
}

// HandlersConfig 处理器配置
type HandlersConfig struct {
	LLMClient      LLMClient
	TTSClient      TTSClient
	OSSClient      OSSClient
	CacheManager   *cache.Manager
}

// NewHandlers 创建处理器集合
func NewHandlers(cfg *HandlersConfig) *Handlers {
	var feedbackCache *cache.FeedbackCache
	var demoAudioCache *cache.DemoAudioCache

	if cfg.CacheManager != nil {
		feedbackCache = cfg.CacheManager.Feedback
		demoAudioCache = cfg.CacheManager.DemoAudio
	}

	return &Handlers{
		FeedbackText:  NewFeedbackTextHandler(cfg.LLMClient, feedbackCache),
		FeedbackAudio: NewFeedbackAudioHandler(cfg.TTSClient, cfg.OSSClient),
		DemoAudio:     NewDemoAudioHandler(cfg.TTSClient, cfg.OSSClient, demoAudioCache),
	}
}

// RegisterToPool 注册所有处理器到工作池
func (h *Handlers) RegisterToPool(pool *async.WorkerPool) {
	pool.RegisterHandler(async.TaskGenerateFeedbackText, h.FeedbackText)
	pool.RegisterHandler(async.TaskGenerateFeedbackAudio, h.FeedbackAudio)
	pool.RegisterHandler(async.TaskGenerateDemoAudio, h.DemoAudio)
}

// NewMockHandlers 创建模拟处理器（用于测试）
func NewMockHandlers() *Handlers {
	return &Handlers{
		FeedbackText:  NewFeedbackTextHandler(NewFallbackLLMClient(), nil),
		FeedbackAudio: NewFeedbackAudioHandler(NewMockTTSClient(), NewMockOSSClient("")),
		DemoAudio:     NewDemoAudioHandler(NewMockTTSClient(), NewMockOSSClient(""), nil),
	}
}
