package qwen

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

// ===================== ChatStream 流式对话实现 =====================

// ChatStream 流式多轮对话（用于 Free Talk 模式）
// 给定完整的对话历史，流式回调每个生成的 token
func (a *QwenAdapter) ChatStream(ctx context.Context, conversationID string, message string) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	return a.qwenClient.chatStream(ctx, conversationID, message)
}

// chatStream 内部实现：使用 openai-go/v3 SDK 的流式 API
func (c *internalClient) chatStream(ctx context.Context, conversationID string, message string) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	stream := c.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model: c.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(message),
		},
		Conversation: responses.ResponseNewParamsConversationUnion{
			OfConversationObject: &responses.ResponseConversationParam{
				ID: conversationID,
			},
		},
	})

	return stream
}
