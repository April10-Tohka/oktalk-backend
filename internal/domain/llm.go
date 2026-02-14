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

	// Close 关闭客户端，释放资源
	Close() error
}

// ChatMessage 对话消息（领域层定义，不依赖任何 SDK）
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}
