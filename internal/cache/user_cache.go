// Package cache 提供用户相关缓存
// 包含用户信息、统计、Token 等缓存
package cache

import (
	"context"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
	"pronunciation-correction-system/internal/model"
)

// UserCache 用户缓存
type UserCache struct {
	commands *redis.Commands
}

// NewUserCache 创建用户缓存
func NewUserCache(commands *redis.Commands) *UserCache {
	return &UserCache{
		commands: commands,
	}
}

// UserStats 用户统计
type UserStats struct {
	TotalEvaluations     int     `json:"total_evaluations"`
	TotalConversations   int     `json:"total_conversations"`
	TotalReports         int     `json:"total_reports"`
	AverageScore         float64 `json:"average_score"`
	TotalDuration        int     `json:"total_duration"` // 秒
	ConsecutiveDays      int     `json:"consecutive_days"`
	LastActiveAt         string  `json:"last_active_at"`
	LastEvaluationAt     string  `json:"last_evaluation_at"`
	LastConversationAt   string  `json:"last_conversation_at"`
	SLevelCount          int     `json:"s_level_count"`
	ALevelCount          int     `json:"a_level_count"`
	BLevelCount          int     `json:"b_level_count"`
	CLevelCount          int     `json:"c_level_count"`
}

// ==================== 用户信息缓存 ====================

// GetProfile 获取用户信息缓存
func (c *UserCache) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	key := redis.Keys.User.Profile(userID)

	var user model.User
	err := c.commands.GetJSON(ctx, key, &user)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// SetProfile 设置用户信息缓存
func (c *UserCache) SetProfile(ctx context.Context, user *model.User) error {
	key := redis.Keys.User.Profile(user.ID)
	return c.commands.SetJSON(ctx, key, user, redis.TTLUserProfile)
}

// DeleteProfile 删除用户信息缓存
func (c *UserCache) DeleteProfile(ctx context.Context, userID string) error {
	key := redis.Keys.User.Profile(userID)
	return c.commands.Del(ctx, key)
}

// ExistsProfile 检查用户信息缓存是否存在
func (c *UserCache) ExistsProfile(ctx context.Context, userID string) (bool, error) {
	key := redis.Keys.User.Profile(userID)
	return c.commands.Exists(ctx, key)
}

// RefreshProfileTTL 刷新用户信息缓存过期时间
func (c *UserCache) RefreshProfileTTL(ctx context.Context, userID string) error {
	key := redis.Keys.User.Profile(userID)
	return c.commands.Expire(ctx, key, redis.TTLUserProfile)
}

// ==================== 用户统计缓存 ====================

// GetStats 获取用户统计缓存
func (c *UserCache) GetStats(ctx context.Context, userID string) (*UserStats, error) {
	key := redis.Keys.User.Stats(userID)

	var stats UserStats
	err := c.commands.GetJSON(ctx, key, &stats)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	return &stats, nil
}

// SetStats 设置用户统计缓存
func (c *UserCache) SetStats(ctx context.Context, userID string, stats *UserStats) error {
	key := redis.Keys.User.Stats(userID)
	return c.commands.SetJSON(ctx, key, stats, redis.TTLUserStats)
}

// DeleteStats 删除用户统计缓存
func (c *UserCache) DeleteStats(ctx context.Context, userID string) error {
	key := redis.Keys.User.Stats(userID)
	return c.commands.Del(ctx, key)
}

// InvalidateStats 使用户统计缓存失效
// 当用户完成评测或对话后调用
func (c *UserCache) InvalidateStats(ctx context.Context, userID string) error {
	return c.DeleteStats(ctx, userID)
}

// ==================== 用户 Token 缓存 ====================

// SetToken 设置用户 Token
func (c *UserCache) SetToken(ctx context.Context, userID, token string, expiration time.Duration) error {
	key := redis.Keys.User.Token(userID)
	return c.commands.Set(ctx, key, token, expiration)
}

// GetToken 获取用户 Token
func (c *UserCache) GetToken(ctx context.Context, userID string) (string, error) {
	key := redis.Keys.User.Token(userID)
	token, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return token, nil
}

// DeleteToken 删除用户 Token
func (c *UserCache) DeleteToken(ctx context.Context, userID string) error {
	key := redis.Keys.User.Token(userID)
	return c.commands.Del(ctx, key)
}

// ValidateToken 验证用户 Token
func (c *UserCache) ValidateToken(ctx context.Context, userID, token string) (bool, error) {
	cachedToken, err := c.GetToken(ctx, userID)
	if err != nil {
		return false, err
	}
	if cachedToken == "" {
		return false, nil
	}
	return cachedToken == token, nil
}

// RefreshToken 刷新 Token 过期时间
func (c *UserCache) RefreshToken(ctx context.Context, userID string, expiration time.Duration) error {
	key := redis.Keys.User.Token(userID)
	return c.commands.Expire(ctx, key, expiration)
}

// ==================== 批量操作 ====================

// DeleteAllUserCache 删除用户所有缓存
func (c *UserCache) DeleteAllUserCache(ctx context.Context, userID string) error {
	// 删除用户信息缓存
	if err := c.DeleteProfile(ctx, userID); err != nil {
		return err
	}

	// 删除用户统计缓存
	if err := c.DeleteStats(ctx, userID); err != nil {
		return err
	}

	// 删除用户 Token
	if err := c.DeleteToken(ctx, userID); err != nil {
		return err
	}

	return nil
}

// GetOrLoadProfile 获取或加载用户信息
// 如果缓存存在则返回缓存，否则调用加载器加载并缓存
func (c *UserCache) GetOrLoadProfile(ctx context.Context, userID string, loader func() (*model.User, error)) (*model.User, bool, error) {
	// 先尝试从缓存获取
	user, err := c.GetProfile(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		return user, true, nil // 缓存命中
	}

	// 缓存未命中，调用加载器
	user, err = loader()
	if err != nil {
		return nil, false, err
	}
	if user == nil {
		return nil, false, nil
	}

	// 缓存加载的数据
	if err := c.SetProfile(ctx, user); err != nil {
		// 缓存失败不影响返回结果
		return user, false, nil
	}

	return user, false, nil
}

// GetOrLoadStats 获取或加载用户统计
func (c *UserCache) GetOrLoadStats(ctx context.Context, userID string, loader func() (*UserStats, error)) (*UserStats, bool, error) {
	// 先尝试从缓存获取
	stats, err := c.GetStats(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if stats != nil {
		return stats, true, nil // 缓存命中
	}

	// 缓存未命中，调用加载器
	stats, err = loader()
	if err != nil {
		return nil, false, err
	}
	if stats == nil {
		return nil, false, nil
	}

	// 缓存加载的数据
	if err := c.SetStats(ctx, userID, stats); err != nil {
		return stats, false, nil
	}

	return stats, false, nil
}
