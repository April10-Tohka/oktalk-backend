// Package handlers 提供具体的任务处理器实现
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"pronunciation-correction-system/internal/async"
	"pronunciation-correction-system/internal/cache"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	GenerateFeedback(ctx context.Context, data map[string]interface{}) (string, error)
}

// FeedbackTextHandler LLM 反馈文本生成处理器
type FeedbackTextHandler struct {
	llmClient     LLMClient
	feedbackCache *cache.FeedbackCache
}

// NewFeedbackTextHandler 创建反馈文本生成处理器
func NewFeedbackTextHandler(llmClient LLMClient, feedbackCache *cache.FeedbackCache) *FeedbackTextHandler {
	return &FeedbackTextHandler{
		llmClient:     llmClient,
		feedbackCache: feedbackCache,
	}
}

// Handle 处理反馈文本生成任务
func (h *FeedbackTextHandler) Handle(ctx context.Context, task *async.EvaluationTask) (*async.TaskResult, error) {
	startTime := time.Now()

	// 提取任务数据
	score := task.GetInt(async.DataKeyScore)
	problemWord := task.GetString(async.DataKeyProblemWord)
	level := task.GetString(async.DataKeyLevel)
	targetText := task.GetString(async.DataKeyTargetText)

	if level == "" {
		level = async.GetFeedbackLevel(score)
	}

	// 1. 检查缓存
	cacheKey := fmt.Sprintf("%d:%s:%s", score, problemWord, level)
	if h.feedbackCache != nil {
		cachedText, err := h.feedbackCache.GetFeedbackText(ctx, score, problemWord, level)
		if err == nil && cachedText != "" {
			return async.NewTaskResult(task.ID, task.Type).
				SetSuccess(map[string]interface{}{
					async.DataKeyFeedbackText: cachedText,
					"from_cache":              true,
					"cache_key":               cacheKey,
				}).
				SetDuration(time.Since(startTime)), nil
		}
	}

	// 2. 调用 LLM 生成反馈文本
	var feedbackText string
	var err error

	if h.llmClient != nil {
		feedbackText, err = h.llmClient.GenerateFeedback(ctx, map[string]interface{}{
			"score":        score,
			"problem_word": problemWord,
			"level":        level,
			"target_text":  targetText,
		})
	}

	// 3. 如果 LLM 失败，使用降级模板
	if err != nil || feedbackText == "" {
		feedbackText = h.getFallbackText(level)
		// 标记为降级
		return async.NewTaskResult(task.ID, task.Type).
			SetSuccess(map[string]interface{}{
				async.DataKeyFeedbackText: feedbackText,
				"fallback":                true,
				"llm_error":               err,
			}).
			SetDuration(time.Since(startTime)), nil
	}

	// 4. 缓存结果
	if h.feedbackCache != nil {
		_ = h.feedbackCache.SetFeedbackText(ctx, score, problemWord, level, feedbackText)
	}

	return async.NewTaskResult(task.ID, task.Type).
		SetSuccess(map[string]interface{}{
			async.DataKeyFeedbackText: feedbackText,
			"from_cache":              false,
			"cache_key":               cacheKey,
		}).
		SetDuration(time.Since(startTime)), nil
}

// getFallbackText 获取降级文本
func (h *FeedbackTextHandler) getFallbackText(level string) string {
	return cache.GetFallbackText(level)
}

// FallbackLLMClient 降级 LLM 客户端（使用模板）
type FallbackLLMClient struct{}

// NewFallbackLLMClient 创建降级 LLM 客户端
func NewFallbackLLMClient() *FallbackLLMClient {
	return &FallbackLLMClient{}
}

// GenerateFeedback 生成反馈（使用模板）
func (c *FallbackLLMClient) GenerateFeedback(ctx context.Context, data map[string]interface{}) (string, error) {
	level, _ := data["level"].(string)
	problemWord, _ := data["problem_word"].(string)

	switch level {
	case async.FeedbackLevelS:
		return "Perfect! Your pronunciation is excellent! Keep up the great work!", nil
	case async.FeedbackLevelA:
		return fmt.Sprintf("Very good! Keep practicing '%s' to make your pronunciation even better.", problemWord), nil
	case async.FeedbackLevelB:
		return fmt.Sprintf("Good try! Let's practice '%s' together. Focus on the vowel sounds.", problemWord), nil
	case async.FeedbackLevelC:
		return fmt.Sprintf("Keep going! Listen to the demo carefully and try '%s' again. Pay attention to stress and rhythm.", problemWord), nil
	default:
		return "Good effort! Keep practicing to improve your pronunciation.", nil
	}
}

// MockLLMClient 模拟 LLM 客户端（用于测试）
type MockLLMClient struct {
	client *redis.Client
}

// NewMockLLMClient 创建模拟 LLM 客户端
func NewMockLLMClient(client *redis.Client) *MockLLMClient {
	return &MockLLMClient{client: client}
}

// GenerateFeedback 生成反馈（模拟）
func (c *MockLLMClient) GenerateFeedback(ctx context.Context, data map[string]interface{}) (string, error) {
	// 模拟 1-2 秒延迟
	time.Sleep(time.Duration(1000+500) * time.Millisecond)

	// 使用降级客户端生成
	fallback := &FallbackLLMClient{}
	return fallback.GenerateFeedback(ctx, data)
}
