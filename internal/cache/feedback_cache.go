// Package cache 提供反馈相关缓存
// 使用 String 结构存储，TTL 7天
// 用于缓存 LLM 生成的反馈文本，相似评分场景可复用
package cache

import (
	"context"
	"encoding/json"

	"pronunciation-correction-system/internal/cache/redis"
)

// FeedbackCache 反馈缓存
type FeedbackCache struct {
	commands *redis.Commands
}

// NewFeedbackCache 创建反馈缓存
func NewFeedbackCache(commands *redis.Commands) *FeedbackCache {
	return &FeedbackCache{
		commands: commands,
	}
}

// FeedbackTextInfo 反馈文本信息
type FeedbackTextInfo struct {
	Text        string `json:"text"`         // 反馈文本
	Level       string `json:"level"`        // 反馈级别 S/A/B/C
	Score       int    `json:"score"`        // 评分
	ProblemWord string `json:"problem_word"` // 问题单词
	AIModel     string `json:"ai_model"`     // AI模型
	GeneratedAt string `json:"generated_at"` // 生成时间
}

// GetFeedbackText 获取反馈文本缓存（基于评分、问题词、级别）
// Key: oktalk:feedback:text:{score}:{problem_word}:{level}
// 用于相似评分场景复用反馈文本
func (c *FeedbackCache) GetFeedbackText(ctx context.Context, score int, problemWord, level string) (string, error) {
	key := redis.Keys.Feedback.Text(score, problemWord, level)
	text, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return text, nil
}

// SetFeedbackText 设置反馈文本缓存（基于评分、问题词、级别）
// TTL: 7天
func (c *FeedbackCache) SetFeedbackText(ctx context.Context, score int, problemWord, level, text string) error {
	key := redis.Keys.Feedback.Text(score, problemWord, level)
	return c.commands.Set(ctx, key, text, redis.TTLFeedbackText)
}

// GetFeedbackTextByEvaluation 获取基于评测ID的反馈文本
// Key: oktalk:feedback:text:{evaluation_id}
func (c *FeedbackCache) GetFeedbackTextByEvaluation(ctx context.Context, evaluationID string) (*FeedbackTextInfo, error) {
	key := redis.Keys.Feedback.TextByEvaluation(evaluationID)

	data, err := c.commands.GetBytes(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var info FeedbackTextInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// SetFeedbackTextByEvaluation 设置基于评测ID的反馈文本
func (c *FeedbackCache) SetFeedbackTextByEvaluation(ctx context.Context, evaluationID string, info *FeedbackTextInfo) error {
	key := redis.Keys.Feedback.TextByEvaluation(evaluationID)

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return c.commands.Set(ctx, key, data, redis.TTLFeedbackText)
}

// GetFeedbackAudioURL 获取反馈音频URL缓存
// Key: oktalk:feedback:audio:{evaluation_id}
func (c *FeedbackCache) GetFeedbackAudioURL(ctx context.Context, evaluationID string) (string, error) {
	key := redis.Keys.Feedback.Audio(evaluationID)
	url, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

// SetFeedbackAudioURL 设置反馈音频URL缓存
func (c *FeedbackCache) SetFeedbackAudioURL(ctx context.Context, evaluationID, url string) error {
	key := redis.Keys.Feedback.Audio(evaluationID)
	return c.commands.Set(ctx, key, url, redis.TTLFeedbackText)
}

// DeleteFeedbackText 删除反馈文本缓存
func (c *FeedbackCache) DeleteFeedbackText(ctx context.Context, score int, problemWord, level string) error {
	key := redis.Keys.Feedback.Text(score, problemWord, level)
	return c.commands.Del(ctx, key)
}

// DeleteFeedbackByEvaluation 删除基于评测ID的反馈缓存
func (c *FeedbackCache) DeleteFeedbackByEvaluation(ctx context.Context, evaluationID string) error {
	textKey := redis.Keys.Feedback.TextByEvaluation(evaluationID)
	audioKey := redis.Keys.Feedback.Audio(evaluationID)
	return c.commands.Del(ctx, textKey, audioKey)
}

// ExistsFeedbackText 检查反馈文本缓存是否存在
func (c *FeedbackCache) ExistsFeedbackText(ctx context.Context, score int, problemWord, level string) (bool, error) {
	key := redis.Keys.Feedback.Text(score, problemWord, level)
	return c.commands.Exists(ctx, key)
}

// GetOrGenerateFeedbackText 获取或生成反馈文本
// 如果缓存存在则返回缓存文本，否则调用生成器生成并缓存
func (c *FeedbackCache) GetOrGenerateFeedbackText(ctx context.Context, score int, problemWord, level string, generator func() (string, error)) (string, bool, error) {
	// 先尝试从缓存获取
	text, err := c.GetFeedbackText(ctx, score, problemWord, level)
	if err != nil {
		return "", false, err
	}
	if text != "" {
		return text, true, nil // 缓存命中
	}

	// 缓存未命中，调用生成器
	text, err = generator()
	if err != nil {
		return "", false, err
	}

	// 缓存生成的文本
	if err := c.SetFeedbackText(ctx, score, problemWord, level, text); err != nil {
		// 缓存失败不影响返回结果
		return text, false, nil
	}

	return text, false, nil
}

// FallbackTemplates 降级模板（当 LLM 失败时使用）
var FallbackTemplates = map[string]string{
	"S": "Perfect! Your pronunciation is excellent!",
	"A": "Very good! Keep practicing to make it even better.",
	"B": "Good try! Let's practice the difficult words together.",
	"C": "Keep going! Listen to the demo and try again.",
}

// GetFallbackText 获取降级文本
func GetFallbackText(level string) string {
	if text, ok := FallbackTemplates[level]; ok {
		return text
	}
	return FallbackTemplates["C"]
}

// SetFeedbackComplete 设置完整的反馈缓存（文本和音频）
func (c *FeedbackCache) SetFeedbackComplete(ctx context.Context, evaluationID string, info *FeedbackTextInfo, audioURL string) error {
	// 设置评测ID关联的反馈信息
	if err := c.SetFeedbackTextByEvaluation(ctx, evaluationID, info); err != nil {
		return err
	}

	// 设置音频URL
	if audioURL != "" {
		if err := c.SetFeedbackAudioURL(ctx, evaluationID, audioURL); err != nil {
			return err
		}
	}

	// 设置基于评分的通用缓存（供相似场景复用）
	if info.ProblemWord != "" {
		if err := c.SetFeedbackText(ctx, info.Score, info.ProblemWord, info.Level, info.Text); err != nil {
			// 通用缓存设置失败不影响主逻辑
			return nil
		}
	}

	return nil
}
