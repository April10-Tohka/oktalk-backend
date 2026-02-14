package aliyun

import (
	"context"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// AliyunASRAdapter 阿里云 FUN-ASR 适配器
// 实现 domain.ASRProvider 接口，将领域层调用转换为阿里云 WebSocket API 调用
type AliyunASRAdapter struct {
	client *internalClient
}

// 编译时检查：确保 AliyunASRAdapter 实现了 domain.ASRProvider 接口
var _ domain.ASRProvider = (*AliyunASRAdapter)(nil)

// NewAliyunASRAdapter 创建阿里云 FUN-ASR 适配器
func NewAliyunASRAdapter(cfg config.AliyunASRConfig) *AliyunASRAdapter {
	return &AliyunASRAdapter{
		client: newInternalClient(cfg),
	}
}

// RecognizeAudio 同步识别音频数据
// 发送完整音频，等待识别完成后返回汇总结果
func (a *AliyunASRAdapter) RecognizeAudio(ctx context.Context, audioData []byte, format string, sampleRate int) (*domain.ASRResult, error) {
	return a.client.recognizeAudio(ctx, audioData, format, sampleRate)
}

// RecognizeAudioStream 流式识别音频数据
// 发送音频的同时实时返回中间结果和最终结果
func (a *AliyunASRAdapter) RecognizeAudioStream(ctx context.Context, audioData []byte, format string, sampleRate int) (<-chan *domain.ASRStreamEvent, error) {
	return a.client.recognizeStream(ctx, audioData, format, sampleRate)
}

// Close 关闭客户端，释放资源
func (a *AliyunASRAdapter) Close() error {
	return a.client.close()
}
