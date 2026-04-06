package repository

import (
	"context"
	"time"
)

// SceneCacheRepository 场景缓存接口，不直接依赖 Redis，方便测试
type SceneCacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
