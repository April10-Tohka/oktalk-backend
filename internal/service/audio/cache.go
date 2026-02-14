// Package audio 提供音频缓存管理
package audio

import (
	"context"
	"time"
)

// CacheManager 音频缓存管理器
type CacheManager struct {
	// TODO: 添加依赖
	// redisClient *redis.Client
	defaultTTL time.Duration
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		defaultTTL: 24 * time.Hour,
	}
}

// GetAudioURL 从缓存获取音频 URL
func (c *CacheManager) GetAudioURL(ctx context.Context, key string) (string, error) {
	// TODO: 实现缓存获取
	return "", nil
}

// SetAudioURL 设置音频 URL 到缓存
func (c *CacheManager) SetAudioURL(ctx context.Context, key, url string, ttl time.Duration) error {
	// TODO: 实现缓存设置
	return nil
}

// GetDemoAudio 获取示范音频缓存
func (c *CacheManager) GetDemoAudio(ctx context.Context, textID string) (*CachedAudio, error) {
	// TODO: 实现示范音频缓存获取
	return nil, nil
}

// SetDemoAudio 设置示范音频缓存
func (c *CacheManager) SetDemoAudio(ctx context.Context, textID string, audio *CachedAudio) error {
	// TODO: 实现示范音频缓存设置
	return nil
}

// InvalidateCache 使缓存失效
func (c *CacheManager) InvalidateCache(ctx context.Context, key string) error {
	// TODO: 实现缓存失效
	return nil
}

// CachedAudio 缓存的音频信息
type CachedAudio struct {
	URL       string    `json:"url"`
	TextID    string    `json:"text_id"`
	Duration  int       `json:"duration"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
