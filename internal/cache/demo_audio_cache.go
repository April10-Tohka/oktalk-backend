// Package cache 提供示范音频URL缓存
// 使用 String 结构存储，TTL 30天
// 用于避免对同一个单词/句子重复调用TTS API
package cache

import (
	"context"

	"pronunciation-correction-system/internal/cache/redis"
)

// DemoAudioCache 示范音频缓存
type DemoAudioCache struct {
	commands *redis.Commands
}

// NewDemoAudioCache 创建示范音频缓存
func NewDemoAudioCache(commands *redis.Commands) *DemoAudioCache {
	return &DemoAudioCache{
		commands: commands,
	}
}

// DemoType 示范音频类型
const (
	DemoTypeWord     = "word"
	DemoTypeSentence = "sentence"
)

// GetWordAudioURL 获取单词示范音频URL
// Key: oktalk:demo:audio:word:{normalized_word}
func (c *DemoAudioCache) GetWordAudioURL(ctx context.Context, word string) (string, error) {
	key := redis.Keys.DemoAudio.Word(word)
	url, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

// SetWordAudioURL 设置单词示范音频URL
// TTL: 30天
func (c *DemoAudioCache) SetWordAudioURL(ctx context.Context, word, url string) error {
	key := redis.Keys.DemoAudio.Word(word)
	return c.commands.Set(ctx, key, url, redis.TTLDemoAudio)
}

// GetSentenceAudioURL 获取句子示范音频URL
// Key: oktalk:demo:audio:sentence:{normalized_sentence}
func (c *DemoAudioCache) GetSentenceAudioURL(ctx context.Context, sentence string) (string, error) {
	key := redis.Keys.DemoAudio.Sentence(sentence)
	url, err := c.commands.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

// SetSentenceAudioURL 设置句子示范音频URL
// TTL: 30天
func (c *DemoAudioCache) SetSentenceAudioURL(ctx context.Context, sentence, url string) error {
	key := redis.Keys.DemoAudio.Sentence(sentence)
	return c.commands.Set(ctx, key, url, redis.TTLDemoAudio)
}

// GetAudioURL 根据类型获取示范音频URL
func (c *DemoAudioCache) GetAudioURL(ctx context.Context, text, demoType string) (string, error) {
	switch demoType {
	case DemoTypeWord:
		return c.GetWordAudioURL(ctx, text)
	case DemoTypeSentence:
		return c.GetSentenceAudioURL(ctx, text)
	default:
		return c.GetWordAudioURL(ctx, text)
	}
}

// SetAudioURL 根据类型设置示范音频URL
func (c *DemoAudioCache) SetAudioURL(ctx context.Context, text, demoType, url string) error {
	switch demoType {
	case DemoTypeWord:
		return c.SetWordAudioURL(ctx, text, url)
	case DemoTypeSentence:
		return c.SetSentenceAudioURL(ctx, text, url)
	default:
		return c.SetWordAudioURL(ctx, text, url)
	}
}

// DeleteWordAudioURL 删除单词示范音频URL缓存
func (c *DemoAudioCache) DeleteWordAudioURL(ctx context.Context, word string) error {
	key := redis.Keys.DemoAudio.Word(word)
	return c.commands.Del(ctx, key)
}

// DeleteSentenceAudioURL 删除句子示范音频URL缓存
func (c *DemoAudioCache) DeleteSentenceAudioURL(ctx context.Context, sentence string) error {
	key := redis.Keys.DemoAudio.Sentence(sentence)
	return c.commands.Del(ctx, key)
}

// ExistsWordAudioURL 检查单词示范音频URL缓存是否存在
func (c *DemoAudioCache) ExistsWordAudioURL(ctx context.Context, word string) (bool, error) {
	key := redis.Keys.DemoAudio.Word(word)
	return c.commands.Exists(ctx, key)
}

// ExistsSentenceAudioURL 检查句子示范音频URL缓存是否存在
func (c *DemoAudioCache) ExistsSentenceAudioURL(ctx context.Context, sentence string) (bool, error) {
	key := redis.Keys.DemoAudio.Sentence(sentence)
	return c.commands.Exists(ctx, key)
}

// ExistsAudioURL 检查示范音频URL缓存是否存在
func (c *DemoAudioCache) ExistsAudioURL(ctx context.Context, text, demoType string) (bool, error) {
	switch demoType {
	case DemoTypeWord:
		return c.ExistsWordAudioURL(ctx, text)
	case DemoTypeSentence:
		return c.ExistsSentenceAudioURL(ctx, text)
	default:
		return c.ExistsWordAudioURL(ctx, text)
	}
}

// GetOrGenerate 获取或生成示范音频URL
// 如果缓存存在则返回缓存URL，否则调用生成器生成并缓存
func (c *DemoAudioCache) GetOrGenerate(ctx context.Context, text, demoType string, generator func() (string, error)) (string, bool, error) {
	// 先尝试从缓存获取
	url, err := c.GetAudioURL(ctx, text, demoType)
	if err != nil {
		return "", false, err
	}
	if url != "" {
		return url, true, nil // 缓存命中
	}

	// 缓存未命中，调用生成器
	url, err = generator()
	if err != nil {
		return "", false, err
	}

	// 缓存生成的URL
	if err := c.SetAudioURL(ctx, text, demoType, url); err != nil {
		// 缓存失败不影响返回结果
		return url, false, nil
	}

	return url, false, nil
}
