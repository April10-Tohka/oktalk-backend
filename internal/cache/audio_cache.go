// Package cache 提供音频相关缓存
// 包含音频URL缓存和通用音频信息缓存
package cache

import (
	"context"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// AudioCache 音频缓存
type AudioCache struct {
	commands *redis.Commands
	ttl      time.Duration
}

// NewAudioCache 创建音频缓存
func NewAudioCache(commands *redis.Commands) *AudioCache {
	return &AudioCache{
		commands: commands,
		ttl:      24 * time.Hour,
	}
}

// AudioInfo 音频信息
type AudioInfo struct {
	URL       string    `json:"url"`
	Duration  int       `json:"duration"` // 毫秒
	Format    string    `json:"format"`   // mp3/wav/etc
	Size      int64     `json:"size"`     // 字节
	CreatedAt time.Time `json:"created_at"`
}

// ==================== 通用音频URL缓存 ====================

// GetAudioURL 获取音频 URL 缓存
func (c *AudioCache) GetAudioURL(ctx context.Context, audioID string) (string, error) {
	key := redis.PrefixSession + "audio:" + audioID // 通用音频key
	url, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

// SetAudioURL 设置音频 URL 缓存
func (c *AudioCache) SetAudioURL(ctx context.Context, audioID, url string) error {
	key := redis.PrefixSession + "audio:" + audioID
	return c.commands.Set(ctx, key, url, c.ttl)
}

// SetAudioURLWithTTL 设置音频 URL 缓存（自定义 TTL）
func (c *AudioCache) SetAudioURLWithTTL(ctx context.Context, audioID, url string, ttl time.Duration) error {
	key := redis.PrefixSession + "audio:" + audioID
	return c.commands.Set(ctx, key, url, ttl)
}

// DeleteAudioURL 删除音频 URL 缓存
func (c *AudioCache) DeleteAudioURL(ctx context.Context, audioID string) error {
	key := redis.PrefixSession + "audio:" + audioID
	return c.commands.Del(ctx, key)
}

// ==================== 音频信息缓存 ====================

// GetAudioInfo 获取音频信息缓存
func (c *AudioCache) GetAudioInfo(ctx context.Context, audioID string) (*AudioInfo, error) {
	key := redis.PrefixSession + "audio:info:" + audioID

	var info AudioInfo
	err := c.commands.GetJSON(ctx, key, &info)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	return &info, nil
}

// SetAudioInfo 设置音频信息缓存
func (c *AudioCache) SetAudioInfo(ctx context.Context, audioID string, info *AudioInfo) error {
	key := redis.PrefixSession + "audio:info:" + audioID
	return c.commands.SetJSON(ctx, key, info, c.ttl)
}

// DeleteAudioInfo 删除音频信息缓存
func (c *AudioCache) DeleteAudioInfo(ctx context.Context, audioID string) error {
	key := redis.PrefixSession + "audio:info:" + audioID
	return c.commands.Del(ctx, key)
}

// ==================== 用户录音缓存 ====================

// GetUserRecordingURL 获取用户录音 URL
func (c *AudioCache) GetUserRecordingURL(ctx context.Context, userID, recordingID string) (string, error) {
	key := redis.PrefixSession + "recording:" + userID + ":" + recordingID
	url, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

// SetUserRecordingURL 设置用户录音 URL
func (c *AudioCache) SetUserRecordingURL(ctx context.Context, userID, recordingID, url string) error {
	key := redis.PrefixSession + "recording:" + userID + ":" + recordingID
	return c.commands.Set(ctx, key, url, c.ttl)
}

// DeleteUserRecordingURL 删除用户录音 URL
func (c *AudioCache) DeleteUserRecordingURL(ctx context.Context, userID, recordingID string) error {
	key := redis.PrefixSession + "recording:" + userID + ":" + recordingID
	return c.commands.Del(ctx, key)
}

// ==================== 批量操作 ====================

// BatchGetAudioURLs 批量获取音频 URL
func (c *AudioCache) BatchGetAudioURLs(ctx context.Context, audioIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, id := range audioIDs {
		url, err := c.GetAudioURL(ctx, id)
		if err != nil {
			return nil, err
		}
		if url != "" {
			result[id] = url
		}
	}
	return result, nil
}

// BatchSetAudioURLs 批量设置音频 URL
func (c *AudioCache) BatchSetAudioURLs(ctx context.Context, urls map[string]string) error {
	for id, url := range urls {
		if err := c.SetAudioURL(ctx, id, url); err != nil {
			return err
		}
	}
	return nil
}

// BatchDeleteAudioURLs 批量删除音频 URL
func (c *AudioCache) BatchDeleteAudioURLs(ctx context.Context, audioIDs []string) error {
	keys := make([]string, len(audioIDs))
	for i, id := range audioIDs {
		keys[i] = redis.PrefixSession + "audio:" + id
	}
	return c.commands.Del(ctx, keys...)
}
