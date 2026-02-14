// Package aliyun 提供阿里云 CosyVoice TTS 语音合成的具体实现
// 基于 DashScope WebSocket API 协议
package aliyun

// ===================== WebSocket 协议数据结构 =====================

// wsEvent WebSocket 事件（请求和响应的统一结构）
type wsEvent struct {
	Header  wsHeader  `json:"header"`
	Payload wsPayload `json:"payload"`
}

// wsHeader 事件头部
type wsHeader struct {
	// Action 请求动作: "run-task", "continue-task", "finish-task"
	Action string `json:"action,omitempty"`

	// TaskID 任务唯一标识
	TaskID string `json:"task_id"`

	// Streaming 流式模式: "duplex"（双工）
	Streaming string `json:"streaming,omitempty"`

	// Event 响应事件类型: "task-started", "result-generated", "task-finished", "task-failed"
	Event string `json:"event,omitempty"`

	// ErrorCode 错误码（仅 task-failed 时有值）
	ErrorCode string `json:"error_code,omitempty"`

	// ErrorMessage 错误信息（仅 task-failed 时有值）
	ErrorMessage string `json:"error_message,omitempty"`

	// Attributes 附加属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// wsPayload 事件负载
type wsPayload struct {
	// 请求字段
	TaskGroup  string   `json:"task_group,omitempty"`
	Task       string   `json:"task,omitempty"`
	Function   string   `json:"function,omitempty"`
	Model      string   `json:"model,omitempty"`
	Parameters wsParams `json:"parameters,omitempty"`
	Input      wsInput  `json:"input"`

	// 响应字段
	Output wsOutput `json:"output,omitempty"`
	Usage  *wsUsage `json:"usage,omitempty"`
}

// wsParams TTS 合成参数
type wsParams struct {
	// TextType 文本类型: "PlainText"
	TextType string `json:"text_type,omitempty"`

	// Voice 音色: "longanyang", "longxiaochun" 等
	Voice string `json:"voice,omitempty"`

	// Format 音频格式: "mp3", "wav", "pcm"
	Format string `json:"format,omitempty"`

	// SampleRate 采样率: 8000, 16000, 22050, 24000, 48000
	SampleRate int `json:"sample_rate,omitempty"`

	// Volume 音量: 0-100
	Volume int `json:"volume,omitempty"`

	// Rate 语速: 0.5-2.0
	Rate float64 `json:"rate,omitempty"`

	// Pitch 音调: 0.5-2.0
	Pitch float64 `json:"pitch,omitempty"`

	// EnableSSML 是否启用 SSML（启用后只允许发送一次 continue-task）
	EnableSSML bool `json:"enable_ssml,omitempty"`
}

// wsInput 输入内容
type wsInput struct {
	// Text 待合成文本（用于 continue-task 指令）
	Text string `json:"text,omitempty"`
}

// wsOutput 输出内容（用于 result-generated 事件）
type wsOutput struct {
	// 部分 TTS 事件可能在此返回额外信息
}

// wsUsage 计费信息
type wsUsage struct {
	// Characters 已消耗字符数
	Characters int `json:"characters,omitempty"`

	// Duration 音频时长（秒）
	Duration int `json:"duration,omitempty"`
}

// ===================== 事件类型常量 =====================

const (
	// 请求动作
	actionRunTask      = "run-task"
	actionContinueTask = "continue-task"
	actionFinishTask   = "finish-task"

	// 响应事件
	eventTaskStarted     = "task-started"
	eventResultGenerated = "result-generated"
	eventTaskFinished    = "task-finished"
	eventTaskFailed      = "task-failed"

	// 流式模式
	streamingDuplex = "duplex"

	// 任务配置
	taskGroupAudio       = "audio"
	taskTTS              = "tts"
	functionSpeechSynth  = "SpeechSynthesizer"
	defaultModel         = "cosyvoice-v3-flash"
	defaultTextType      = "PlainText"

	// 地域 WebSocket 地址
	wsURLBeijing    = "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
	wsURLSingapore  = "wss://dashscope-intl.aliyuncs.com/api-ws/v1/inference/"
)
