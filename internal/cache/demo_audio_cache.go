package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ===================== DemoAudioCache =====================

// DemoAudioCache 示范音频缓存（全局共享）
type DemoAudioCache struct {
	rdb *redis.Client
}

// NewDemoAudioCache 创建 DemoAudioCache
func NewDemoAudioCache(rdb *redis.Client) *DemoAudioCache {
	return &DemoAudioCache{rdb: rdb}
}

// GetWordDemoAudio 获取单词示范音频 URL
func (c *DemoAudioCache) GetWordDemoAudio(ctx context.Context, word, voice string) (string, bool, error) {
	key := fmt.Sprintf(KeyDemoAudioWord, word, voice)
	return c.getString(ctx, key)
}

// SetWordDemoAudio 写入单词示范音频 URL
func (c *DemoAudioCache) SetWordDemoAudio(ctx context.Context, word, voice, url string) error {
	key := fmt.Sprintf(KeyDemoAudioWord, word, voice)
	return c.rdb.Set(ctx, key, url, TTLDemoAudio).Err()
}

// GetSentenceDemoAudio 获取句子示范音频 URL（text 自动 MD5Hash）
func (c *DemoAudioCache) GetSentenceDemoAudio(ctx context.Context, text, voice string) (string, bool, error) {
	hash := MD5Hash(text)
	key := fmt.Sprintf(KeyDemoAudioSentence, hash, voice)
	return c.getString(ctx, key)
}

// SetSentenceDemoAudio 写入句子示范音频 URL（text 自动 MD5Hash）
func (c *DemoAudioCache) SetSentenceDemoAudio(ctx context.Context, text, voice, url string) error {
	hash := MD5Hash(text)
	key := fmt.Sprintf(KeyDemoAudioSentence, hash, voice)
	return c.rdb.Set(ctx, key, url, TTLDemoAudio).Err()
}

// getString 获取字符串类型缓存
func (c *DemoAudioCache) getString(ctx context.Context, key string) (string, bool, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}
