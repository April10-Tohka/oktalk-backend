// Package aliyun 提供阿里云 FUN-ASR 实时语音识别的具体实现
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
	// Action 请求动作: "run-task", "finish-task"
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
	TaskGroup  string    `json:"task_group,omitempty"`
	Task       string    `json:"task,omitempty"`
	Function   string    `json:"function,omitempty"`
	Model      string    `json:"model,omitempty"`
	Parameters wsParams  `json:"parameters,omitempty"`
	Input      wsInput   `json:"input,omitempty"`

	// 响应字段
	Output wsOutput `json:"output,omitempty"`
	Usage  *wsUsage `json:"usage,omitempty"`
}

// wsParams 任务参数
type wsParams struct {
	// Format 音频格式: "wav", "pcm", "mp3"
	Format string `json:"format,omitempty"`

	// SampleRate 采样率: 16000
	SampleRate int `json:"sample_rate,omitempty"`

	// VocabularyID 自定义热词表 ID（可选）
	VocabularyID string `json:"vocabulary_id,omitempty"`

	// DisfluencyRemovalEnabled 是否启用语气词过滤
	DisfluencyRemovalEnabled bool `json:"disfluency_removal_enabled,omitempty"`
}

// wsInput 输入（run-task 和 finish-task 中使用，一般为空结构）
type wsInput struct{}

// wsOutput 输出结果
type wsOutput struct {
	Sentence wsSentence `json:"sentence,omitempty"`
}

// wsSentence 句子级识别结果
type wsSentence struct {
	// BeginTime 句子开始时间（毫秒）
	BeginTime int64 `json:"begin_time"`

	// EndTime 句子结束时间（毫秒），中间结果时可能为 nil
	EndTime *int64 `json:"end_time"`

	// Text 识别文本
	Text string `json:"text"`

	// Words 单词级结果列表
	Words []wsWord `json:"words,omitempty"`
}

// wsWord 单词级识别结果
type wsWord struct {
	// BeginTime 单词开始时间（毫秒）
	BeginTime int64 `json:"begin_time"`

	// EndTime 单词结束时间（毫秒）
	EndTime *int64 `json:"end_time"`

	// Text 单词文本
	Text string `json:"text"`

	// Punctuation 标点符号
	Punctuation string `json:"punctuation,omitempty"`
}

// wsUsage 计费信息
type wsUsage struct {
	// Duration 音频时长（秒）
	Duration int `json:"duration"`
}

// ===================== 事件类型常量 =====================

const (
	// 请求动作
	actionRunTask    = "run-task"
	actionFinishTask = "finish-task"

	// 响应事件
	eventTaskStarted    = "task-started"
	eventResultGenerate = "result-generated"
	eventTaskFinished   = "task-finished"
	eventTaskFailed     = "task-failed"

	// 流式模式
	streamingDuplex = "duplex"

	// 任务配置
	taskGroupAudio    = "audio"
	taskASR           = "asr"
	functionRecognize = "recognition"
	modelFunASR       = "paraformer-realtime-v2"

	// 默认配置
	defaultSendChunkSize = 3200 // 每次发送音频数据块大小（字节），对应 100ms 的 16kHz 16bit PCM
	defaultSendInterval  = 100  // 发送间隔（毫秒），模拟实时音频流
)
