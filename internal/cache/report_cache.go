package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ===================== ReportCache =====================

// ReportCache 报告缓存操作
type ReportCache struct {
	rdb *redis.Client
}

// NewReportCache 创建 ReportCache
func NewReportCache(rdb *redis.Client) *ReportCache {
	return &ReportCache{rdb: rdb}
}

// SetReportResult 写入报告结果（任意类型存储为 JSON）
func (c *ReportCache) SetReportResult(ctx context.Context, reportID string, result interface{}) error {
	key := fmt.Sprintf(KeyReportResult, reportID)
	return SetJSON(ctx, c.rdb, key, result, TTLReportResult)
}

// GetReportResult 获取报告结果
func (c *ReportCache) GetReportResult(ctx context.Context, reportID string) (map[string]interface{}, bool, error) {
	key := fmt.Sprintf(KeyReportResult, reportID)
	result, found, err := GetJSON[map[string]interface{}](ctx, c.rdb, key)
	if err != nil {
		return nil, false, err
	}
	if !found || result == nil {
		return nil, false, nil
	}
	return *result, true, nil
}

// SetReportHistory 写入报告历史分页
func (c *ReportCache) SetReportHistory(ctx context.Context, userID string, page int, data *PagedList) error {
	key := fmt.Sprintf(KeyReportHistory, userID, page)
	return SetJSON(ctx, c.rdb, key, data, TTLReportHistory)
}

// GetReportHistory 获取报告历史分页
func (c *ReportCache) GetReportHistory(ctx context.Context, userID string, page int) (*PagedList, bool, error) {
	key := fmt.Sprintf(KeyReportHistory, userID, page)
	return GetJSON[PagedList](ctx, c.rdb, key)
}

// InvalidateReportHistory 删除该用户所有页的报告历史缓存
func (c *ReportCache) InvalidateReportHistory(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("report:history:%s:*", userID)
	return deleteByPattern(ctx, c.rdb, pattern)
}
