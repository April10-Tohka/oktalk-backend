package aliyun

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// AliyunTTSAdapter 阿里云 CosyVoice TTS 适配器
// 实现 domain.TTSProvider 接口，将领域层调用转换为阿里云 WebSocket API 调用
type AliyunTTSAdapter struct {
	client *internalClient
}

// 编译时检查：确保 AliyunTTSAdapter 实现了 domain.TTSProvider 接口
var _ domain.TTSProvider = (*AliyunTTSAdapter)(nil)

// NewAliyunTTSAdapter 创建阿里云 CosyVoice TTS 适配器
func NewAliyunTTSAdapter(cfg config.AliyunTTSConfig) *AliyunTTSAdapter {
	return &AliyunTTSAdapter{
		client: newInternalClient(cfg),
	}
}

// Synthesize 合成语音（同步方式，返回完整音频数据）
// 将单段文本发送给 CosyVoice，等待合成完成后返回完整音频
func (a *AliyunTTSAdapter) Synthesize(ctx context.Context, text string, options *domain.SynthesizeOptions) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	audioData, err := a.client.synthesize(ctx, []string{text}, options)
	if err != nil {
		return nil, fmt.Errorf("aliyun tts synthesize failed: %w", err)
	}

	return audioData, nil
}

// SynthesizeToFile 合成语音并保存到本地文件
func (a *AliyunTTSAdapter) SynthesizeToFile(ctx context.Context, text string, outputPath string, options *domain.SynthesizeOptions) error {
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}
	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// 先合成获取音频数据
	audioData, err := a.Synthesize(ctx, text, options)
	if err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(outputPath, audioData, 0644); err != nil {
		return fmt.Errorf("write audio file failed: %w", err)
	}

	slog.Info("[AliyunTTS] Audio saved to file",
		"path", outputPath,
		"bytes", len(audioData),
	)

	return nil
}

// SynthesizeStream 流式合成语音（实时返回音频片段）
// 音频块通过 audioChan 实时推送，合成完成后关闭 channel
func (a *AliyunTTSAdapter) SynthesizeStream(ctx context.Context, text string, options *domain.SynthesizeOptions, audioChan chan<- []byte) error {
	if text == "" {
		close(audioChan)
		return fmt.Errorf("text cannot be empty")
	}

	err := a.client.synthesizeStream(ctx, []string{text}, options, audioChan)
	if err != nil {
		return fmt.Errorf("aliyun tts stream failed: %w", err)
	}

	return nil
}

// SynthesizeMultiple 批量合成（多段文本拼接为一个音频）
// 通过 WebSocket 的 continue-task 机制连续发送多段文本
func (a *AliyunTTSAdapter) SynthesizeMultiple(ctx context.Context, texts []string, options *domain.SynthesizeOptions) ([]byte, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// 过滤空文本
	validTexts := make([]string, 0, len(texts))
	for _, t := range texts {
		if t != "" {
			validTexts = append(validTexts, t)
		}
	}
	if len(validTexts) == 0 {
		return nil, fmt.Errorf("all texts are empty")
	}

	audioData, err := a.client.synthesize(ctx, validTexts, options)
	if err != nil {
		return nil, fmt.Errorf("aliyun tts synthesize multiple failed: %w", err)
	}

	return audioData, nil
}

// Close 关闭客户端，释放资源
func (a *AliyunTTSAdapter) Close() error {
	return a.client.close()
}
