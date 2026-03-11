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

	// ConnectTTS 建立流式 TTS 长连接（用于 Free Talk 模式）
	// 返回 TTSStreamer，支持在同一连接上执行多个合成任务（每轮对话一个任务）。
	// 使用流程：RunTask() → FeedText() * N → FinishTask() → 等待 TaskDone() → 再次 RunTask()...
	// 参数:
	//   - ctx: 上下文，控制整个连接的生命周期
	//   - options: 合成选项（音色、格式、采样率等）
	// 返回:
	//   - TTSStreamer: 流式 TTS 操作接口
	//   - error: 连接或初始化错误
	ConnectTTS(ctx context.Context, options *SynthesizeOptions) (TTSStreamer, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// TTSStreamer 流式 TTS 连接接口（会话级）
// 用于 Free Talk 模式下的多轮文本推送和音频接收。
// 每个 TTSStreamer 对应一个 WebSocket 连接，可以在上面执行多个合成任务。
type TTSStreamer interface {
	// RunTask 启动新的合成任务（发送 run-task 指令）
	// 每轮对话开始时调用，阻塞直到收到 task-started 或超时
	RunTask(ctx context.Context) error

	// FeedText 向当前任务推送文本片段（发送 continue-task 指令）
	// 在 LLM 流式生成过程中，每积累一段文本就调用一次
	FeedText(ctx context.Context, text string) error

	// FinishTask 通知当前任务文本发送完毕（发送 finish-task 指令）
	// LLM 生成结束后调用，CosyVoice 将继续返回剩余音频直到 task-finished
	FinishTask(ctx context.Context) error

	// AudioChan 返回接收合成音频数据的通道
	// 音频以 PCM/MP3 二进制块形式推送，通道在 Close() 时关闭
	AudioChan() <-chan []byte

	// TaskDone 返回一个通道，当前任务完成（task-finished）时关闭
	// 每次 RunTask 后会重新创建此通道
	TaskDone() <-chan struct{}

	// Close 关闭 WebSocket 连接并释放所有资源
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
