package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ===================== TTS 缓存结构 =====================

// SynthesizeOptions TTS 合成选项（用于缓存 Key 生成）
type SynthesizeOptions struct {
	Format     string  `json:"format"`
	SampleRate int     `json:"sample_rate"`
	Rate       float64 `json:"rate"`
}

// ===================== TTSCache =====================

// TTSCache TTS 音频缓存
type TTSCache struct {
	rdb *redis.Client
}

// NewTTSCache 创建 TTSCache
func NewTTSCache(rdb *redis.Client) *TTSCache {
	return &TTSCache{rdb: rdb}
}

// GetTTSAudio 获取 TTS 音频缓存
// text 自动 normalizeText + MD5Hash
// opts 序列化为 "{Format}_{SampleRate}_{Rate}" 后 MD5Hash
func (c *TTSCache) GetTTSAudio(ctx context.Context, text, voice string, opts *SynthesizeOptions) (string, bool, error) {
	key := c.buildKey(text, voice, opts)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}

// SetTTSAudio 写入 TTS 音频缓存
func (c *TTSCache) SetTTSAudio(ctx context.Context, text, voice string, opts *SynthesizeOptions, ossURL string) error {
	key := c.buildKey(text, voice, opts)
	return c.rdb.Set(ctx, key, ossURL, TTLTTSAudio).Err()
}

// buildKey 构建 TTS 缓存 Key
func (c *TTSCache) buildKey(text, voice string, opts *SynthesizeOptions) string {
	textHash := MD5Hash(normalizeText(text))
	optsHash := MD5Hash(formatOptions(opts))
	return fmt.Sprintf(KeyTTSAudio, textHash, voice, optsHash)
}

// normalizeText 文本规范化：去首尾空格、转小写、合并连续空格
func normalizeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	// 合并连续空格
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}

// formatOptions 将选项序列化为字符串
func formatOptions(opts *SynthesizeOptions) string {
	if opts == nil {
		return "mp3_22050_1.0"
	}
	return fmt.Sprintf("%s_%d_%.1f", opts.Format, opts.SampleRate, opts.Rate)
}
