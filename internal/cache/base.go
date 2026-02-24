package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ===================== 通用缓存操作（泛型封装） =====================

// GetJSON 获取 JSON 缓存，未命中返回 nil, false
func GetJSON[T any](ctx context.Context, rdb *redis.Client, key string) (*T, bool, error) {
	data, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, false, err
	}
	return &val, true, nil
}

// SetJSON 设置 JSON 缓存，ttl=0 表示永久
func SetJSON[T any](ctx context.Context, rdb *redis.Client, key string, val T, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	if ttl > 0 {
		return rdb.Set(ctx, key, data, ttl).Err()
	}
	// ttl=0 不设置过期时间
	return rdb.Set(ctx, key, data, 0).Err()
}

// Del 删除一个或多个 Key
func Del(ctx context.Context, rdb *redis.Client, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rdb.Del(ctx, keys...).Err()
}

// RPush 追加到 List（用于会话记录）
func RPush(ctx context.Context, rdb *redis.Client, key string, val interface{}, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	if err := rdb.RPush(ctx, key, data).Err(); err != nil {
		return err
	}
	if ttl > 0 {
		return rdb.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// ===================== ZSet 操作 =====================

// ZAdd 添加成员到 Sorted Set
func ZAdd(ctx context.Context, rdb *redis.Client, key string, member string, score float64) error {
	return rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRem 从 Sorted Set 移除成员
func ZRem(ctx context.Context, rdb *redis.Client, key string, member string) error {
	return rdb.ZRem(ctx, key, member).Err()
}

// ZScan 扫描 Sorted Set（分批迭代，避免阻塞）
func ZScan(ctx context.Context, rdb *redis.Client, key string, cursor uint64, count int64) ([]string, uint64, error) {
	return rdb.ZScan(ctx, key, cursor, "*", count).Result()
}
