package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// ===================== Eval 缓存结构 =====================

// EvalResult 评测结果
type EvalResult struct {
	EvalID         string  `json:"eval_id"`
	UserID         string  `json:"user_id"`
	OverallScore   float64 `json:"overall_score"`
	AccuracyScore  float64 `json:"accuracy_score"`
	FluencyScore   float64 `json:"fluency_score"`
	IntegrityScore float64 `json:"integrity_score"`
	FeedbackLevel  string  `json:"feedback_level"`
	FeedbackText   string  `json:"feedback_text"`
	AudioURL       string  `json:"audio_url,omitempty"`
	DemoAudioURL   string  `json:"demo_audio_url,omitempty"`
	CreatedAt      int64   `json:"created_at"`
}

// ===================== EvalCache =====================

// EvalCache 评测缓存操作
type EvalCache struct {
	rdb *redis.Client
}

// NewEvalCache 创建 EvalCache
func NewEvalCache(rdb *redis.Client) *EvalCache {
	return &EvalCache{rdb: rdb}
}

// SetEvalResult 写入评测结果
func (c *EvalCache) SetEvalResult(ctx context.Context, evalID string, result *EvalResult) error {
	key := fmt.Sprintf(KeyEvalResult, evalID)
	return SetJSON(ctx, c.rdb, key, result, TTLEvalResult)
}

// GetEvalResult 获取评测结果
func (c *EvalCache) GetEvalResult(ctx context.Context, evalID string) (*EvalResult, bool, error) {
	key := fmt.Sprintf(KeyEvalResult, evalID)
	return GetJSON[EvalResult](ctx, c.rdb, key)
}

// SetEvalHistory 写入评测历史分页
func (c *EvalCache) SetEvalHistory(ctx context.Context, userID string, page int, data *PagedList) error {
	key := fmt.Sprintf(KeyEvalHistory, userID, page)
	return SetJSON(ctx, c.rdb, key, data, TTLHistoryList)
}

// GetEvalHistory 获取评测历史分页
func (c *EvalCache) GetEvalHistory(ctx context.Context, userID string, page int) (*PagedList, bool, error) {
	key := fmt.Sprintf(KeyEvalHistory, userID, page)
	return GetJSON[PagedList](ctx, c.rdb, key)
}

// InvalidateEvalHistory 删除该用户所有页的评测历史缓存
func (c *EvalCache) InvalidateEvalHistory(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("evaluate:history:%s:*", userID)
	return deleteByPattern(ctx, c.rdb, pattern)
}

// deleteByPattern 使用 SCAN 批量删除匹配的 Key
func deleteByPattern(ctx context.Context, rdb *redis.Client, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				slog.Warn("batch delete keys failed", "pattern", pattern, "error", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
