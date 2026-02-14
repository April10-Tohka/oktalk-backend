// Package qwen 提供通义千问 LLM 的具体实现
// 通过 OpenAI 兼容 SDK（github.com/openai/openai-go）与 DashScope API 通信
// 此包内的所有类型对外部（Service 层）不可见，
// 仅通过 QwenAdapter 实现 domain.LLMProvider 接口对外暴露
package qwen

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"pronunciation-correction-system/internal/config"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// internalClient 通义千问内部 SDK 客户端
// 封装 openai-go SDK，负责与 DashScope API 通信
type internalClient struct {
	model  string         // 模型名称，如 qwen-turbo, qwen-plus
	client *openai.Client // openai-go SDK 客户端实例
}

// newInternalClient 根据配置创建内部客户端
// 使用 openai-go SDK 的 OpenAI 兼容模式连接 DashScope
func newInternalClient(cfg config.QwenConfig) *internalClient {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// 如果配置了 BaseURL，则使用自定义地址（DashScope 兼容模式）
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	log.Printf("[Qwen] Client initialized, model=%s, baseURL=%s", cfg.Model, cfg.BaseURL)

	return &internalClient{
		model:  cfg.Model,
		client: &client,
	}
}

// chat 调用 Chat Completions API 进行对话补全
// 参数:
//   - ctx: 上下文，支持超时和取消
//   - req: 对话请求，包含消息列表和可选参数
//
// 返回:
//   - *chatResponse: 对话响应结果
//   - error: 错误信息，SDK 错误会被捕获并包装为业务友好的错误
func (c *internalClient) chat(ctx context.Context, req *chatRequest) (*chatResponse, error) {
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
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	// 确定模型：优先使用请求中指定的模型，否则使用默认模型
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

	// 调用 SDK
	completion, err := c.client.Chat.Completions.New(ctx, params)
	elapsed := time.Since(start)

	if err != nil {
		// 使用 errors.As 捕获 SDK 特定错误类型
		var apiErr *openai.Error
		if errors.As(err, &apiErr) {
			log.Printf("[Qwen] API error: status=%d, message=%s, elapsed=%v",
				apiErr.StatusCode, apiErr.Message, elapsed)
			return nil, fmt.Errorf("qwen api error (status %d): %s", apiErr.StatusCode, apiErr.Message)
		}

		// 检查是否为 context 超时或取消
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[Qwen] Request timeout, elapsed=%v", elapsed)
			return nil, fmt.Errorf("qwen request timeout after %v: %w", elapsed, err)
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("[Qwen] Request canceled, elapsed=%v", elapsed)
			return nil, fmt.Errorf("qwen request canceled: %w", err)
		}

		// 其他未知错误
		log.Printf("[Qwen] Unknown error: %v, elapsed=%v", err, elapsed)
		return nil, fmt.Errorf("qwen chat failed: %w", err)
	}

	// 校验响应
	if len(completion.Choices) == 0 {
		log.Printf("[Qwen] Empty response choices, model=%s, elapsed=%v", model, elapsed)
		return nil, fmt.Errorf("qwen returned empty response")
	}

	// 记录成功日志（不含敏感信息）
	log.Printf("[Qwen] Chat completed: model=%s, promptTokens=%d, completionTokens=%d, totalTokens=%d, elapsed=%v",
		completion.Model,
		completion.Usage.PromptTokens,
		completion.Usage.CompletionTokens,
		completion.Usage.TotalTokens,
		elapsed,
	)

	// 转换为内部响应结构
	choices := make([]chatChoice, len(completion.Choices))
	for i, c := range completion.Choices {
		choices[i] = chatChoice{
			Index:        int(c.Index),
			FinishReason: string(c.FinishReason),
			Message: chatMessage{
				Role:    string(c.Message.Role),
				Content: c.Message.Content,
			},
		}
	}

	return &chatResponse{
		ID:      completion.ID,
		Model:   completion.Model,
		Choices: choices,
		Usage: chatUsage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}, nil
}

// close 关闭客户端，释放资源
func (c *internalClient) close() error {
	log.Println("[Qwen] Client closed")
	return nil
}
