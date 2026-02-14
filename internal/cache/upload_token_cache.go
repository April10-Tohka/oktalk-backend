// Package cache 提供临时上传令牌缓存
// 使用 String 结构存储 JSON 数据，TTL 5分钟
// 用于前端获取上传凭证和验证上传请求合法性
package cache

import (
	"context"
	"encoding/json"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// UploadTokenCache 临时上传令牌缓存
type UploadTokenCache struct {
	commands *redis.Commands
}

// NewUploadTokenCache 创建临时上传令牌缓存
func NewUploadTokenCache(commands *redis.Commands) *UploadTokenCache {
	return &UploadTokenCache{
		commands: commands,
	}
}

// UploadTokenInfo 上传令牌信息
type UploadTokenInfo struct {
	UserID      string    `json:"user_id"`
	MaxSize     int64     `json:"max_size"`      // 最大文件大小（字节）
	AllowedType string    `json:"allowed_type"`  // 允许的文件类型
	Purpose     string    `json:"purpose"`       // 用途: audio/image/document
	ExpiresAt   time.Time `json:"expires_at"`    // 过期时间
	CreatedAt   time.Time `json:"created_at"`    // 创建时间
	Metadata    string    `json:"metadata"`      // 额外元数据
}

// DefaultUploadTokenInfo 默认上传令牌信息
func DefaultUploadTokenInfo(userID string) *UploadTokenInfo {
	now := time.Now()
	return &UploadTokenInfo{
		UserID:      userID,
		MaxSize:     10 * 1024 * 1024, // 10MB
		AllowedType: "audio/*",
		Purpose:     "audio",
		ExpiresAt:   now.Add(redis.TTLUploadToken),
		CreatedAt:   now,
	}
}

// SetToken 设置上传令牌
// Key: oktalk:temp:upload:{token}
// TTL: 5分钟
func (c *UploadTokenCache) SetToken(ctx context.Context, token string, info *UploadTokenInfo) error {
	key := redis.Keys.Temp.UploadToken(token)

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return c.commands.Set(ctx, key, data, redis.TTLUploadToken)
}

// GetToken 获取上传令牌信息
func (c *UploadTokenCache) GetToken(ctx context.Context, token string) (*UploadTokenInfo, error) {
	key := redis.Keys.Temp.UploadToken(token)

	data, err := c.commands.GetBytes(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var info UploadTokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// DeleteToken 删除上传令牌
func (c *UploadTokenCache) DeleteToken(ctx context.Context, token string) error {
	key := redis.Keys.Temp.UploadToken(token)
	return c.commands.Del(ctx, key)
}

// ExistsToken 检查上传令牌是否存在
func (c *UploadTokenCache) ExistsToken(ctx context.Context, token string) (bool, error) {
	key := redis.Keys.Temp.UploadToken(token)
	return c.commands.Exists(ctx, key)
}

// ValidateToken 验证上传令牌
// 检查令牌是否存在且未过期，验证后自动删除（一次性令牌）
func (c *UploadTokenCache) ValidateToken(ctx context.Context, token, userID string) (*UploadTokenInfo, error) {
	info, err := c.GetToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil // 令牌不存在
	}

	// 验证用户ID
	if info.UserID != userID {
		return nil, nil // 用户不匹配
	}

	// 检查过期时间
	if time.Now().After(info.ExpiresAt) {
		// 已过期，删除令牌
		_ = c.DeleteToken(ctx, token)
		return nil, nil
	}

	// 验证成功，删除令牌（一次性使用）
	_ = c.DeleteToken(ctx, token)

	return info, nil
}

// ValidateAndKeepToken 验证上传令牌但不删除
// 用于需要多次验证的场景
func (c *UploadTokenCache) ValidateAndKeepToken(ctx context.Context, token, userID string) (*UploadTokenInfo, error) {
	info, err := c.GetToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	// 验证用户ID
	if info.UserID != userID {
		return nil, nil
	}

	// 检查过期时间
	if time.Now().After(info.ExpiresAt) {
		_ = c.DeleteToken(ctx, token)
		return nil, nil
	}

	return info, nil
}

// CreateToken 创建新的上传令牌
// 返回令牌字符串，需要调用者生成唯一的 token 字符串
func (c *UploadTokenCache) CreateToken(ctx context.Context, token string, info *UploadTokenInfo) error {
	// 设置过期时间
	if info.ExpiresAt.IsZero() {
		info.ExpiresAt = time.Now().Add(redis.TTLUploadToken)
	}
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}

	return c.SetToken(ctx, token, info)
}

// CreateTokenWithDefaults 使用默认配置创建上传令牌
func (c *UploadTokenCache) CreateTokenWithDefaults(ctx context.Context, token, userID string) error {
	info := DefaultUploadTokenInfo(userID)
	return c.SetToken(ctx, token, info)
}

// CreateAudioUploadToken 创建音频上传令牌
func (c *UploadTokenCache) CreateAudioUploadToken(ctx context.Context, token, userID string, maxSize int64) error {
	now := time.Now()
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 默认 10MB
	}

	info := &UploadTokenInfo{
		UserID:      userID,
		MaxSize:     maxSize,
		AllowedType: "audio/*",
		Purpose:     "audio",
		ExpiresAt:   now.Add(redis.TTLUploadToken),
		CreatedAt:   now,
	}

	return c.SetToken(ctx, token, info)
}

// GetTokenTTL 获取令牌剩余有效时间
func (c *UploadTokenCache) GetTokenTTL(ctx context.Context, token string) (time.Duration, error) {
	key := redis.Keys.Temp.UploadToken(token)
	return c.commands.TTL(ctx, key)
}

// RefreshToken 刷新令牌过期时间
func (c *UploadTokenCache) RefreshToken(ctx context.Context, token string) error {
	key := redis.Keys.Temp.UploadToken(token)

	// 检查是否存在
	exists, err := c.commands.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return c.commands.Expire(ctx, key, redis.TTLUploadToken)
}
