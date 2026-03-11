// Package domain 定义核心业务接口
// 所有接口方法只使用 Go 原生类型，严禁出现任何第三方 SDK 结构体
package domain

import "context"

// LLMProvider 大语言模型服务提供者接口
// 用于生成发音反馈文本、对话等 AI 功能
type LLMProvider interface {
	// Chat 单轮对话：给定系统提示词和用户消息，返回 AI 生成的文本
	Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error)

	// ChatWithHistory 多轮对话：给定完整的对话历史，返回 AI 生成的文本
	ChatWithHistory(ctx context.Context, messages []ChatMessage) (string, error)

	// ChatStream 流式多轮对话（用于 Free Talk 模式）
	// 给定完整的对话历史，流式回调每个生成的 token。
	// onToken: 每生成一个 token 回调一次，传入增量文本片段
	// 方法阻塞直到生成完成或出错
	ChatStream(ctx context.Context, messages []ChatMessage, onToken func(token string)) error

	// Close 关闭客户端，释放资源
	Close() error
}

// ChatMessage 对话消息（领域层定义，不依赖任何 SDK）
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}
