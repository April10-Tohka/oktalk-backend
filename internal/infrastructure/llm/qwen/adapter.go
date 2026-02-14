package qwen

import (
	"context"
	"fmt"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// QwenAdapter 通义千问适配器
// 实现 domain.LLMProvider 接口，将领域层调用转换为通义千问 SDK 调用
// 职责：参数转换（domain 类型 → SDK 内部类型），不包含任何 SDK 细节
type QwenAdapter struct {
	qwenClient *internalClient
}

// 编译时检查：确保 QwenAdapter 实现了 domain.LLMProvider 接口
var _ domain.LLMProvider = (*QwenAdapter)(nil)

// NewQwenAdapter 创建通义千问适配器
// 接收 QwenConfig 配置，内部初始化 openai-go SDK 客户端
func NewQwenAdapter(cfg config.QwenConfig) *QwenAdapter {
	return &QwenAdapter{
		qwenClient: newInternalClient(cfg),
	}
}

// Chat 单轮对话
// 给定系统提示词和用户消息，返回 AI 生成的文本
func (a *QwenAdapter) Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	req := &chatRequest{
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
	}

	resp, err := a.qwenClient.chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("qwen chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("qwen returned empty response")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatWithHistory 多轮对话
// 给定完整的对话历史（包含 system/user/assistant 消息），返回 AI 生成的文本
func (a *QwenAdapter) ChatWithHistory(ctx context.Context, messages []domain.ChatMessage) (string, error) {
	// 将领域层 ChatMessage 转换为内部 chatMessage
	internalMessages := make([]chatMessage, len(messages))
	for i, msg := range messages {
		internalMessages[i] = chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := &chatRequest{
		Messages: internalMessages,
	}

	resp, err := a.qwenClient.chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("qwen chat with history failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("qwen returned empty response")
	}

	return resp.Choices[0].Message.Content, nil
}

// Close 关闭客户端，释放资源
func (a *QwenAdapter) Close() error {
	return a.qwenClient.close()
}
