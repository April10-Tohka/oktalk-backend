package cache

import (
	"context"
	"fmt"
	"math"

	"github.com/redis/go-redis/v9"
)

// ===================== LLMCache =====================

// LLMCache LLM 结果缓存
type LLMCache struct {
	rdb *redis.Client
}

// NewLLMCache 创建 LLMCache
func NewLLMCache(rdb *redis.Client) *LLMCache {
	return &LLMCache{rdb: rdb}
}

// GetFeedbackCache 获取 LLM 反馈缓存
// 内部实现：分数区间化 scoreRange = Round(score/5)*5
// S 级 直接返回 "", false, nil（不走缓存）
// C 级 对 wordOrSentence 做 MD5Hash
func (c *LLMCache) GetFeedbackCache(ctx context.Context, level string, score float64, wordOrSentence string) (string, bool, error) {
	// S 级不走缓存
	if level == "S" {
		return "", false, nil
	}

	key := c.buildFeedbackKey(level, score, wordOrSentence)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}

// SetFeedbackCache 写入 LLM 反馈缓存
func (c *LLMCache) SetFeedbackCache(ctx context.Context, level string, score float64, wordOrSentence, text string) error {
	// S 级不写缓存
	if level == "S" {
		return nil
	}

	key := c.buildFeedbackKey(level, score, wordOrSentence)
	return c.rdb.Set(ctx, key, text, TTLLLMFeedback).Err()
}

// GetReportCache 获取 LLM 报告缓存
func (c *LLMCache) GetReportCache(ctx context.Context, userID, period, statsHash string) (map[string]interface{}, bool, error) {
	key := fmt.Sprintf(KeyLLMReport, userID, period, statsHash)
	result, found, err := GetJSON[map[string]interface{}](ctx, c.rdb, key)
	if err != nil {
		return nil, false, err
	}
	if !found || result == nil {
		return nil, false, nil
	}
	return *result, true, nil
}

// SetReportCache 写入 LLM 报告缓存
func (c *LLMCache) SetReportCache(ctx context.Context, userID, period, statsHash string, data map[string]interface{}) error {
	key := fmt.Sprintf(KeyLLMReport, userID, period, statsHash)
	return SetJSON(ctx, c.rdb, key, data, TTLLLMReport)
}

// buildFeedbackKey 构建反馈缓存 Key
// 分数区间化：Round(score/5)*5
// C 级对 wordOrSentence 做 MD5Hash
func (c *LLMCache) buildFeedbackKey(level string, score float64, wordOrSentence string) string {
	scoreRange := int(math.Round(score/5) * 5)

	wordKey := wordOrSentence
	if level == "C" || len(wordOrSentence) > 50 {
		wordKey = MD5Hash(wordOrSentence)
	}

	return fmt.Sprintf(KeyLLMFeedback, level, scoreRange, wordKey)
}
