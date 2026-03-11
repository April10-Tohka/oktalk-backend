// Package freetalk 实现 Free Talk 模式的 WebSocket 实时语音对话
// 与现有"一句一句"模式不同，Free Talk 使用流式 ASR + LLM + TTS 实现连续对话
package freetalk

// ===================== Text Frame 类型常量 =====================

const (
	// MsgTypeLLMToken LLM 流式 token（后端 → App）
	// 每个 token 作为一条 text frame 推送，App 可实时展示打字效果
	MsgTypeLLMToken = "llm_token"

	// MsgTypeTurnEnd 本轮 AI 回复结束（后端 → App）
	// App 收到后可恢复录音
	MsgTypeTurnEnd = "turn_end"

	// MsgTypeError 错误通知（后端 → App）
	// 包含 code 和 message 字段
	MsgTypeError = "error"

	// MsgTypeASRText ASR 最终识别文本（后端 → App）
	// 用于在 App 端展示用户说的话
	MsgTypeASRText = "asr_text"

	// MsgTypeASRPartial ASR 中间识别文本（后端 → App）
	// 用于实时展示识别过程
	MsgTypeASRPartial = "asr_partial"
)

// ===================== App → 后端（Text Frame）=====================

// IncomingMessage App 发来的 Text Frame 结构
type IncomingMessage struct {
	// Type 消息类型: "start" / "stop"
	// "start": 开始/恢复 Free Talk 会话
	// "stop": 结束 Free Talk 会话
	Type string `json:"type"`

	// ConversationID 会话 ID（start 时必传）
	ConversationID string `json:"conversation_id,omitempty"`
}

// ===================== 后端 → App（Text Frame）=====================

// OutgoingMessage 后端推给 App 的 Text Frame 结构
type OutgoingMessage struct {
	// Type 消息类型（见上方常量）
	Type string `json:"type"`

	// Text 文本内容
	// - llm_token: LLM 生成的增量 token
	// - asr_text: ASR 最终识别文本
	// - asr_partial: ASR 中间识别文本
	// - turn_end: 空
	Text string `json:"text,omitempty"`

	// Code 错误码（仅 error 时使用）
	Code string `json:"code,omitempty"`

	// Message 错误信息（仅 error 时使用）
	Message string `json:"message,omitempty"`
}

// ===================== Binary Frame =====================
// Binary Frame 携带 PCM 裸音频数据，无任何包装：
// - App → 后端：用户录音 PCM，16kHz 单声道 16bit
// - 后端 → App：TTS 合成音频（格式由配置决定，默认 PCM）

// ===================== WebSocket 内部消息类型 =====================

// wsMessage 内部 WebSocket 消息封装
// 用于 writerGoroutine 串行化所有写操作
type wsMessage struct {
	// messageType websocket.TextMessage 或 websocket.BinaryMessage
	messageType int

	// data 消息内容（JSON 或二进制音频）
	data []byte
}
