package qwen

import (
	"context"
	"fmt"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

// QwenAdapter 通义千问适配器
// 实现 domain.LLMProvider 接口，将领域层调用转换为通义千问 SDK 调用
// 职责：参数转换（domain 类型 → SDK 内部类型），不包含任何 SDK 细节
type QwenAdapter struct {
	qwenClient *internalClient
	model      string         // 模型名称，如 kimi-k2.5
	client     *openai.Client // openai-go SDK 客户端实例
}

// 编译时检查：确保 QwenAdapter 实现了 domain.LLMProvider 接口
var _ domain.LLMProvider = (*QwenAdapter)(nil)

// NewQwenAdapter 创建通义千问适配器
// 接收 QwenConfig 配置，内部初始化 openai-go SDK 客户端
func NewQwenAdapter(cfg config.QwenConfig) *QwenAdapter {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)
	logger.Info("[Qwen] Client initialized", "model", cfg.Model, "baseURL", cfg.BaseURL)

	return &QwenAdapter{
		qwenClient: newInternalClient(cfg),
		model:      cfg.Model,
		client:     &client,
	}
}

// Chat 单轮对话
// 给定系统提示词和用户消息，返回 AI 生成的文本
func (a *QwenAdapter) Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	resp, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Instructions: openai.String(systemPrompt),
	})
	if err != nil {
		return "", fmt.Errorf("qwen chat failed: %w", err)
	}
	return resp.OutputText(), nil
}

// ChatStream 流式对话
// 给定系统提示词和用户消息，返回 AI 生成的文本流
func (a *QwenAdapter) ChatStream(ctx context.Context, systemPrompt string, userMessage string) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	stream := a.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Instructions: openai.String(systemPrompt),
	})
	return stream
}

// NewConversation 创建新对话，设置系统提示词，返回对话 ID
func (a *QwenAdapter) NewConversation(ctx context.Context, systemPrompt string) (string, error) {
	items := []responses.ResponseInputItemUnionParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Role: responses.EasyInputMessageRoleSystem,
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: openai.String(systemPrompt),
				},
			},
		},
	}
	conv, err := a.client.Conversations.New(ctx, conversations.ConversationNewParams{
		Items: items,
	})
	if err != nil {
		return "", err
	}
	return conv.ID, nil
}

// ConversationChatStream 基于对话 ID 进行流式对话
func (a *QwenAdapter) ConversationChatStream(ctx context.Context, conversationID string, userMessage string) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	stream := a.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Conversation: responses.ResponseNewParamsConversationUnion{
			OfConversationObject: &responses.ResponseConversationParam{
				ID: conversationID,
			},
		},
	})
	return stream
}

// Close 关闭客户端，释放资源
func (a *QwenAdapter) Close() error {
	return a.qwenClient.close()
}
