package qwen

// ===================== 内部数据结构（供 adapter 和 client 使用） =====================
// 这些结构体隔离了 openai-go SDK 的具体类型，使 adapter 层无需直接依赖 SDK

// chatRequest 对话请求参数
type chatRequest struct {
	Model       string        `json:"model"`                 // 模型名称
	Messages    []chatMessage `json:"messages"`              // 消息列表
	Temperature *float64      `json:"temperature,omitempty"` // 采样温度，0.0~2.0
	MaxTokens   int           `json:"max_tokens,omitempty"`  // 最大生成 token 数
	TopP        *float64      `json:"top_p,omitempty"`       // 核采样概率，0.0~1.0
}

// chatMessage 消息
type chatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // 消息内容
}

// chatResponse 对话响应
type chatResponse struct {
	ID      string       `json:"id"`      // 请求 ID
	Model   string       `json:"model"`   // 实际使用的模型
	Choices []chatChoice `json:"choices"` // 候选回复列表
	Usage   chatUsage    `json:"usage"`   // token 使用量
}

// chatChoice 候选回复
type chatChoice struct {
	Index        int         `json:"index"`         // 候选序号
	Message      chatMessage `json:"message"`       // 回复消息
	FinishReason string      `json:"finish_reason"` // 结束原因: stop, length, etc.
}

// chatUsage token 使用量统计
type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入 token 数
	CompletionTokens int `json:"completion_tokens"` // 输出 token 数
	TotalTokens      int `json:"total_tokens"`      // 总 token 数
}
