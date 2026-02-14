// Package handlers 提供示范音频生成处理器
package handlers

import (
	"context"
	"fmt"
	"time"

	"pronunciation-correction-system/internal/async"
	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/cache/redis"
)

// DemoAudioHandler 示范音频生成处理器
type DemoAudioHandler struct {
	ttsClient      TTSClient
	ossClient      OSSClient
	demoAudioCache *cache.DemoAudioCache
}

// NewDemoAudioHandler 创建示范音频生成处理器
func NewDemoAudioHandler(ttsClient TTSClient, ossClient OSSClient, demoAudioCache *cache.DemoAudioCache) *DemoAudioHandler {
	return &DemoAudioHandler{
		ttsClient:      ttsClient,
		ossClient:      ossClient,
		demoAudioCache: demoAudioCache,
	}
}

// Handle 处理示范音频生成任务
func (h *DemoAudioHandler) Handle(ctx context.Context, task *async.EvaluationTask) (*async.TaskResult, error) {
	startTime := time.Now()

	// 提取任务数据
	demoText := task.GetString(async.DataKeyDemoText)
	demoType := task.GetString(async.DataKeyDemoType)

	if demoText == "" {
		return nil, fmt.Errorf("demo text is required")
	}

	if demoType == "" {
		demoType = async.DemoTypeWord
	}

	// 标准化文本用于缓存 key
	normalizedText := redis.NormalizeText(demoText)
	cacheKey := fmt.Sprintf("%s:%s", demoType, normalizedText)

	// 1. 检查缓存
	if h.demoAudioCache != nil {
		var cachedURL string
		var err error

		switch demoType {
		case async.DemoTypeWord:
			cachedURL, err = h.demoAudioCache.GetWordAudioURL(ctx, demoText)
		case async.DemoTypeSentence:
			cachedURL, err = h.demoAudioCache.GetSentenceAudioURL(ctx, demoText)
		}

		if err == nil && cachedURL != "" {
			return async.NewTaskResult(task.ID, task.Type).
				SetSuccess(map[string]interface{}{
					async.DataKeyDemoAudioURL: cachedURL,
					"from_cache":              true,
					"cache_key":               cacheKey,
				}).
				SetDuration(time.Since(startTime)), nil
		}
	}

	// 2. 调用 TTS 生成音频
	var audioData []byte
	var err error

	if h.ttsClient != nil {
		audioData, err = h.ttsClient.Synthesize(ctx, demoText)
		if err != nil {
			return nil, fmt.Errorf("TTS synthesis failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("TTS client not configured")
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("TTS returned empty audio data")
	}

	// 3. 上传到 OSS
	var audioURL string
	if h.ossClient != nil {
		objectKey := fmt.Sprintf("demo/%s/%s.mp3", demoType, normalizedText)
		audioURL, err = h.ossClient.Upload(ctx, audioData, objectKey)
		if err != nil {
			return nil, fmt.Errorf("OSS upload failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("OSS client not configured")
	}

	// 4. 缓存 URL
	if h.demoAudioCache != nil {
		switch demoType {
		case async.DemoTypeWord:
			_ = h.demoAudioCache.SetWordAudioURL(ctx, demoText, audioURL)
		case async.DemoTypeSentence:
			_ = h.demoAudioCache.SetSentenceAudioURL(ctx, demoText, audioURL)
		}
	}

	return async.NewTaskResult(task.ID, task.Type).
		SetSuccess(map[string]interface{}{
			async.DataKeyDemoAudioURL: audioURL,
			"from_cache":              false,
			"cache_key":               cacheKey,
			async.DataKeyDuration:     len(audioData) / 32000, // 估算时长
		}).
		SetDuration(time.Since(startTime)), nil
}
