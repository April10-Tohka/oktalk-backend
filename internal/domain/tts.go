package domain

import "context"

// ===================== TTS 语音合成接口 =====================

// TTSProvider 语音合成服务提供者接口（业务层抽象）
// 所有方法仅使用 Go 原生类型，不依赖任何第三方 SDK
type TTSProvider interface {
	// Synthesize 合成语音（同步方式，返回完整音频数据）
	// text: 待合成文本
	// options: 合成选项，传 nil 则使用默认参数
	Synthesize(ctx context.Context, text string, options *SynthesizeOptions) ([]byte, error)

	// SynthesizeToFile 合成语音并保存到本地文件
	SynthesizeToFile(ctx context.Context, text string, outputPath string, options *SynthesizeOptions) error

	// SynthesizeStream 流式合成（实时返回音频片段）
	// 音频块通过 audioChan 实时推送，合成完成后关闭 channel
	SynthesizeStream(ctx context.Context, text string, options *SynthesizeOptions, audioChan chan<- []byte) error

	// SynthesizeMultiple 批量合成（多段文本拼接为一个音频）
	SynthesizeMultiple(ctx context.Context, texts []string, options *SynthesizeOptions) ([]byte, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// ===================== 合成选项 =====================

// SynthesizeOptions 合成选项
type SynthesizeOptions struct {
	Voice      string  // 音色：longanyang, longxiaochun, longwan 等
	Format     string  // 格式：mp3, wav, pcm
	SampleRate int     // 采样率：8000, 16000, 22050, 24000, 48000
	Volume     int     // 音量：0-100，默认 50
	Rate       float64 // 语速：0.5-2.0，默认 1.0
	Pitch      float64 // 音调：0.5-2.0，默认 1.0
}

// DefaultSynthesizeOptions 返回默认合成选项
func DefaultSynthesizeOptions() *SynthesizeOptions {
	return &SynthesizeOptions{
		Voice:      "longanyang",
		Format:     "mp3",
		SampleRate: 22050,
		Volume:     50,
		Rate:       1.0,
		Pitch:      1.0,
	}
}

// MergeDefaults 将当前选项与默认值合并
// 未设置的字段使用默认值填充
func (o *SynthesizeOptions) MergeDefaults(defaults *SynthesizeOptions) *SynthesizeOptions {
	if o == nil {
		return defaults
	}
	if defaults == nil {
		return o
	}

	merged := *o
	if merged.Voice == "" {
		merged.Voice = defaults.Voice
	}
	if merged.Format == "" {
		merged.Format = defaults.Format
	}
	if merged.SampleRate == 0 {
		merged.SampleRate = defaults.SampleRate
	}
	if merged.Volume == 0 {
		merged.Volume = defaults.Volume
	}
	if merged.Rate == 0 {
		merged.Rate = defaults.Rate
	}
	if merged.Pitch == 0 {
		merged.Pitch = defaults.Pitch
	}
	return &merged
}
