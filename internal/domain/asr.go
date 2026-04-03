// Package domain 定义核心业务接口
package domain

import (
	"context"

	"github.com/gorilla/websocket"
)

// ASRProvider 语音识别服务提供者接口
// 用于将音频转换为文本（Automatic Speech Recognition）
type ASRProvider interface {
	// RecognizeAudio 识别音频数据（同步方式）
	// 发送完整音频数据，等待所有识别结果返回后汇总为最终文本
	// 参数:
	//   - ctx: 上下文，支持超时和取消
	//   - audioData: 音频二进制数据
	// 返回:
	//   - string: 识别结果
	//   - error: 错误信息
	RecognizeAudio(ctx context.Context, audioData []byte) (string, error)

	// RecognizeAudioStream 流式识别音频数据（异步方式）
	// 发送音频数据的同时实时接收识别的中间结果和最终结果
	// 参数:
	//   - ctx: 上下文，支持超时和取消
	//   - audioData: 音频二进制数据
	//   - format: 音频格式，如 "wav", "pcm", "mp3"
	//   - sampleRate: 采样率，如 16000
	// 返回:
	//   - <-chan *ASRStreamEvent: 流式事件通道，包含中间结果和最终结果
	//   - error: 连接或初始化错误
	RecognizeAudioStream(ctx context.Context, audioData []byte, format string, sampleRate int) (<-chan *ASRStreamEvent, error)

	// ConnectASR 建立流式 ASR 长连接（用于 Free Talk 模式）
	// 与 RecognizeAudioStream 不同，此方法不一次性传入音频，而是返回一个 AudioSender，
	// 允许调用方持续推送实时音频数据，同时通过 eventCh 接收识别结果（含 VAD 断句）。
	// 连接在整个 Free Talk 会话期间保持不断开。
	// 参数:
	//   - ctx: 上下文，支持超时和取消
	//   - format: 音频格式，如 "pcm"
	//   - sampleRate: 采样率，如 16000
	// 返回:
	//   - AudioSender: 用于持续发送实时音频数据的发送器
	//   - <-chan *ASRStreamEvent: 流式事件通道，包含中间结果和最终结果（sentence_end=true 表示断句）
	//   - error: 连接或初始化错误
	ConnectASR(ctx context.Context) (*websocket.Conn, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// AudioSender 音频发送器接口（会话级长连接）
// 用于 Free Talk 模式下持续向 ASR 引擎发送实时音频数据
type AudioSender interface {
	// SendAudio 发送一块音频数据
	SendAudio(data []byte) error

	// Close 发送 finish-task 并关闭连接
	Close() error
}

// ASRResult 语音识别最终结果（同步模式）
type ASRResult struct {
	Text     string          `json:"text"`     // 完整识别文本
	Duration int             `json:"duration"` // 音频时长（秒）
	Words    []ASRWordResult `json:"words"`    // 单词级结果（如果支持）
}

// ASRWordResult 单词级识别结果
type ASRWordResult struct {
	Text        string `json:"text"`        // 单词文本
	BeginTime   int64  `json:"begin_time"`  // 开始时间（毫秒）
	EndTime     int64  `json:"end_time"`    // 结束时间（毫秒）
	Punctuation string `json:"punctuation"` // 标点符号
}

// ASRStreamEvent 流式识别事件
type ASRStreamEvent struct {
	// Type 事件类型: "partial"（中间结果）, "final"（最终结果）, "error"（错误）
	Type string `json:"type"`

	// Text 当前识别文本
	Text string `json:"text"`

	// BeginTime 句子开始时间（毫秒）
	BeginTime int64 `json:"begin_time"`

	// EndTime 句子结束时间（毫秒），中间结果可能为 0
	EndTime int64 `json:"end_time"`

	// Words 单词级结果（仅最终结果包含完整信息）
	Words []ASRWordResult `json:"words,omitempty"`

	// Duration 音频总时长（秒），仅在最终事件中有值
	Duration int `json:"duration,omitempty"`

	// Error 错误信息（仅 Type="error" 时有值）
	Error error `json:"-"`
}
