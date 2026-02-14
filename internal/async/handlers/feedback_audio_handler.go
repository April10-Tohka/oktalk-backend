// Package handlers 提供 TTS 反馈音频生成处理器
package handlers

import (
	"context"
	"fmt"
	"time"

	"pronunciation-correction-system/internal/async"
)

// TTSClient TTS 客户端接口
type TTSClient interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}

// OSSClient OSS 客户端接口
type OSSClient interface {
	Upload(ctx context.Context, data []byte, objectKey string) (string, error)
}

// FeedbackAudioHandler TTS 反馈语音生成处理器
type FeedbackAudioHandler struct {
	ttsClient TTSClient
	ossClient OSSClient
}

// NewFeedbackAudioHandler 创建反馈音频生成处理器
func NewFeedbackAudioHandler(ttsClient TTSClient, ossClient OSSClient) *FeedbackAudioHandler {
	return &FeedbackAudioHandler{
		ttsClient: ttsClient,
		ossClient: ossClient,
	}
}

// Handle 处理反馈音频生成任务
func (h *FeedbackAudioHandler) Handle(ctx context.Context, task *async.EvaluationTask) (*async.TaskResult, error) {
	startTime := time.Now()

	// 提取任务数据
	evaluationID := task.GetString(async.DataKeyEvaluationID)
	feedbackText := task.GetString(async.DataKeyFeedbackText)

	if feedbackText == "" {
		return nil, fmt.Errorf("feedback text is required")
	}

	// 1. 调用 TTS 生成音频
	var audioData []byte
	var err error

	if h.ttsClient != nil {
		audioData, err = h.ttsClient.Synthesize(ctx, feedbackText)
		if err != nil {
			return nil, fmt.Errorf("TTS synthesis failed: %w", err)
		}
	} else {
		// 没有 TTS 客户端，返回空结果
		return async.NewTaskResult(task.ID, task.Type).
			SetSuccess(map[string]interface{}{
				async.DataKeyFeedbackAudioURL: "",
				"skipped":                     true,
				"reason":                      "TTS client not configured",
			}).
			SetDuration(time.Since(startTime)), nil
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("TTS returned empty audio data")
	}

	// 2. 上传到 OSS
	var audioURL string
	if h.ossClient != nil {
		objectKey := fmt.Sprintf("feedback/%s.mp3", evaluationID)
		audioURL, err = h.ossClient.Upload(ctx, audioData, objectKey)
		if err != nil {
			return nil, fmt.Errorf("OSS upload failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("OSS client not configured")
	}

	return async.NewTaskResult(task.ID, task.Type).
		SetSuccess(map[string]interface{}{
			async.DataKeyFeedbackAudioURL: audioURL,
			async.DataKeyDuration:         len(audioData) / 32000, // 估算时长（假设 32kHz）
		}).
		SetDuration(time.Since(startTime)), nil
}

// MockTTSClient 模拟 TTS 客户端（用于测试）
type MockTTSClient struct{}

// NewMockTTSClient 创建模拟 TTS 客户端
func NewMockTTSClient() *MockTTSClient {
	return &MockTTSClient{}
}

// Synthesize 合成语音（模拟）
func (c *MockTTSClient) Synthesize(ctx context.Context, text string) ([]byte, error) {
	// 模拟 1-2 秒延迟
	time.Sleep(time.Duration(1000+500) * time.Millisecond)

	// 返回模拟音频数据
	return []byte("mock_audio_data_" + text), nil
}

// MockOSSClient 模拟 OSS 客户端（用于测试）
type MockOSSClient struct {
	baseURL string
}

// NewMockOSSClient 创建模拟 OSS 客户端
func NewMockOSSClient(baseURL string) *MockOSSClient {
	if baseURL == "" {
		baseURL = "https://cdn.oktalk.com"
	}
	return &MockOSSClient{baseURL: baseURL}
}

// Upload 上传文件（模拟）
func (c *MockOSSClient) Upload(ctx context.Context, data []byte, objectKey string) (string, error) {
	// 模拟上传延迟
	time.Sleep(500 * time.Millisecond)

	// 返回模拟 URL
	return fmt.Sprintf("%s/%s", c.baseURL, objectKey), nil
}
