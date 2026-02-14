// Package cache 提供用户配额缓存
// 使用 String 结构存储计数器，TTL 为当天结束时间
// 用于限制用户每日评测次数
package cache

import (
	"context"
	"fmt"
	"strconv"

	"pronunciation-correction-system/internal/cache/redis"
)

// QuotaCache 用户配额缓存
type QuotaCache struct {
	commands     *redis.Commands
	defaultQuota int // 默认每日配额
}

// NewQuotaCache 创建用户配额缓存
func NewQuotaCache(commands *redis.Commands, defaultQuota int) *QuotaCache {
	if defaultQuota <= 0 {
		defaultQuota = 50 // 默认50次/天
	}
	return &QuotaCache{
		commands:     commands,
		defaultQuota: defaultQuota,
	}
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	UserID    string `json:"user_id"`
	Used      int    `json:"used"`       // 已使用次数
	Remaining int    `json:"remaining"`  // 剩余次数
	Total     int    `json:"total"`      // 总配额
	Date      string `json:"date"`       // 日期
}

// GetUsed 获取用户今日已使用配额
func (c *QuotaCache) GetUsed(ctx context.Context, userID string) (int, error) {
	key := redis.Keys.User.QuotaToday(userID)
	value, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return 0, nil
		}
		return 0, err
	}
	used, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse quota value: %w", err)
	}
	return used, nil
}

// Increment 增加用户配额使用（+1）
// 如果 key 不存在，会自动创建并设置过期时间为当天结束
func (c *QuotaCache) Increment(ctx context.Context, userID string) (int, error) {
	key := redis.Keys.User.QuotaToday(userID)

	// 增加计数
	newValue, err := c.commands.Incr(ctx, key)
	if err != nil {
		return 0, err
	}

	// 如果是第一次设置，设置过期时间为当天结束
	if newValue == 1 {
		ttl := redis.CalculateTodayRemainingTTL()
		if err := c.commands.Expire(ctx, key, ttl); err != nil {
			// 设置过期时间失败不影响计数
			return int(newValue), nil
		}
	}

	return int(newValue), nil
}

// IncrementBy 增加用户配额使用（指定数量）
func (c *QuotaCache) IncrementBy(ctx context.Context, userID string, delta int) (int, error) {
	key := redis.Keys.User.QuotaToday(userID)

	// 先检查是否存在
	exists, _ := c.commands.Exists(ctx, key)

	// 增加计数
	newValue, err := c.commands.IncrBy(ctx, key, int64(delta))
	if err != nil {
		return 0, err
	}

	// 如果是新创建的 key，设置过期时间
	if !exists {
		ttl := redis.CalculateTodayRemainingTTL()
		if err := c.commands.Expire(ctx, key, ttl); err != nil {
			return int(newValue), nil
		}
	}

	return int(newValue), nil
}

// CheckAndIncrement 检查配额并增加使用
// 如果剩余配额不足，返回 false 和当前使用量
// 如果有剩余配额，增加使用量并返回 true 和新的使用量
func (c *QuotaCache) CheckAndIncrement(ctx context.Context, userID string, quota int) (bool, int, error) {
	if quota <= 0 {
		quota = c.defaultQuota
	}

	// 获取当前使用量
	used, err := c.GetUsed(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	// 检查配额
	if used >= quota {
		return false, used, nil
	}

	// 增加使用量
	newUsed, err := c.Increment(ctx, userID)
	if err != nil {
		return false, used, err
	}

	// 再次检查（防止并发超限）
	if newUsed > quota {
		return false, newUsed, nil
	}

	return true, newUsed, nil
}

// GetQuotaInfo 获取用户配额信息
func (c *QuotaCache) GetQuotaInfo(ctx context.Context, userID string, quota int) (*QuotaInfo, error) {
	if quota <= 0 {
		quota = c.defaultQuota
	}

	used, err := c.GetUsed(ctx, userID)
	if err != nil {
		return nil, err
	}

	remaining := quota - used
	if remaining < 0 {
		remaining = 0
	}

	return &QuotaInfo{
		UserID:    userID,
		Used:      used,
		Remaining: remaining,
		Total:     quota,
	}, nil
}

// SetQuota 直接设置用户配额使用量
// 用于管理员调整配额或恢复配额
func (c *QuotaCache) SetQuota(ctx context.Context, userID string, used int) error {
	key := redis.Keys.User.QuotaToday(userID)

	// 设置值
	err := c.commands.Set(ctx, key, strconv.Itoa(used), redis.CalculateTodayRemainingTTL())
	if err != nil {
		return err
	}

	return nil
}

// ResetQuota 重置用户今日配额
func (c *QuotaCache) ResetQuota(ctx context.Context, userID string) error {
	key := redis.Keys.User.QuotaToday(userID)
	return c.commands.Del(ctx, key)
}

// IsExceeded 检查用户是否超过配额
func (c *QuotaCache) IsExceeded(ctx context.Context, userID string, quota int) (bool, error) {
	if quota <= 0 {
		quota = c.defaultQuota
	}

	used, err := c.GetUsed(ctx, userID)
	if err != nil {
		return false, err
	}

	return used >= quota, nil
}

// GetRemaining 获取用户剩余配额
func (c *QuotaCache) GetRemaining(ctx context.Context, userID string, quota int) (int, error) {
	if quota <= 0 {
		quota = c.defaultQuota
	}

	used, err := c.GetUsed(ctx, userID)
	if err != nil {
		return 0, err
	}

	remaining := quota - used
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Decrement 减少用户配额使用（-1）
// 用于取消操作时回退配额
func (c *QuotaCache) Decrement(ctx context.Context, userID string) (int, error) {
	key := redis.Keys.User.QuotaToday(userID)

	newValue, err := c.commands.Decr(ctx, key)
	if err != nil {
		return 0, err
	}

	// 防止负数
	if newValue < 0 {
		if err := c.commands.Set(ctx, key, "0", redis.CalculateTodayRemainingTTL()); err != nil {
			return 0, err
		}
		return 0, nil
	}

	return int(newValue), nil
}
