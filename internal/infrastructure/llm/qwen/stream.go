package qwen

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"pronunciation-correction-system/internal/domain"

	"github.com/openai/openai-go/v3"
)

// ===================== ChatStream 流式对话实现 =====================

// ChatStream 流式多轮对话（用于 Free Talk 模式）
// 给定完整的对话历史，流式回调每个生成的 token
func (a *QwenAdapter) ChatStream(ctx context.Context, messages []domain.ChatMessage, onToken func(token string)) error {
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

	return a.qwenClient.chatStream(ctx, req, onToken)
}

// chatStream 内部实现：使用 openai-go/v3 SDK 的流式 API
func (c *internalClient) chatStream(ctx context.Context, req *chatRequest, onToken func(string)) error {
	start := time.Now()

	// 构建 SDK 消息列表
	sdkMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			sdkMessages = append(sdkMessages, openai.SystemMessage(msg.Content))
		case "user":
			sdkMessages = append(sdkMessages, openai.UserMessage(msg.Content))
		case "assistant":
			sdkMessages = append(sdkMessages, openai.AssistantMessage(msg.Content))
		default:
			return fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	// 确定模型
	model := c.model
	if req.Model != "" {
		model = req.Model
	}

	// 构建 SDK 请求参数
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: sdkMessages,
	}

	// 设置可选参数
	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.TopP != nil {
		params.TopP = openai.Float(*req.TopP)
	}

	// 使用流式 API
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	var totalTokens int
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				totalTokens++
				onToken(delta)
			}
		}
	}

	elapsed := time.Since(start)

	if err := stream.Err(); err != nil {
		slog.Error("[Qwen] Stream error",
			"error", err,
			"model", model,
			"elapsed", elapsed.String(),
		)
		return fmt.Errorf("qwen stream failed: %w", err)
	}

	slog.Info("[Qwen] Stream completed",
		"model", model,
		"tokens", totalTokens,
		"elapsed", elapsed.String(),
	)

	return nil
}
