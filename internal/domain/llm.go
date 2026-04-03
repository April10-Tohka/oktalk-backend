// Package domain 定义核心业务接口
// 所有接口方法只使用 Go 原生类型，严禁出现任何第三方 SDK 结构体
package domain

import (
	"context"

	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

// LLMProvider 大语言模型服务提供者接口
// 用于生成发音反馈文本、对话等 AI 功能
type LLMProvider interface {
	// Chat 单轮对话：给定系统提示词和用户消息，返回 AI 生成的文本
	Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error)

	// ChatStream 流式对话
	// 给定系统提示词和用户消息，返回 AI 生成的文本流
	ChatStream(ctx context.Context, systemPrompt string, userMessage string) *ssestream.Stream[responses.ResponseStreamEventUnion]

	// NewConversation 创建新对话，设置系统提示词
	NewConversation(ctx context.Context, systemPrompt string) (string, error)

	// ConversationChatStream 基于对话 ID 进行流式对话
	ConversationChatStream(ctx context.Context, conversationID string, userMessage string) *ssestream.Stream[responses.ResponseStreamEventUnion]

	// Close 关闭客户端，释放资源
	Close() error
}

// ChatMessage 对话消息（领域层定义，不依赖任何 SDK）
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

type LLMChunk struct {
	Text   string
	IsDone bool // true 表示本轮生成结束
}
