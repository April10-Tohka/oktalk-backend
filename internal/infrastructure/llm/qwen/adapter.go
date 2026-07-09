package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
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
	params := responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Instructions: openai.String(systemPrompt),
		Reasoning: shared.ReasoningParam{
			Summary: "concise",
			Effort:  shared.ReasoningEffortNone,
		},
	}
	params.SetExtraFields(map[string]any{
		"enable_thinking": false, // 改成 false，关闭思考模式，显著降低延迟
	})
	resp, err := a.client.Responses.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("qwen chat failed: %w", err)
	}
	return resp.OutputText(), nil
}

// ChatStream 流式对话
// 给定系统提示词和用户消息，返回 AI 生成的文本流
func (a *QwenAdapter) ChatStream(ctx context.Context, systemPrompt string, userMessage string) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	params := responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Instructions: openai.String(systemPrompt),
		Reasoning: shared.ReasoningParam{
			Summary: "concise",
			Effort:  shared.ReasoningEffortNone,
		},
	}
	params.SetExtraFields(map[string]any{
		"enable_thinking": false, // 改成 false，关闭思考模式，显著降低延迟
	})
	stream := a.client.Responses.NewStreaming(ctx, params)
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
	params := responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		},
		Conversation: responses.ResponseNewParamsConversationUnion{
			OfConversationObject: &responses.ResponseConversationParam{
				ID: conversationID,
			},
		},
		Reasoning: shared.ReasoningParam{
			Summary: "concise",
			Effort:  shared.ReasoningEffortNone,
		},
	}
	params.SetExtraFields(map[string]any{
		"enable_thinking": false, // 改成 false，关闭思考模式，显著降低延迟
	})
	stream := a.client.Responses.NewStreaming(ctx, params)
	return stream
}

func (a *QwenAdapter) ChatHistoryStream(ctx context.Context, messages []domain.Message) *ssestream.Stream[openai.ChatCompletionChunk] {
	// 转换为 openai-go 的格式
	MessagesParams := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system":
			MessagesParams = append(MessagesParams, openai.SystemMessage(m.Content))
		case "user":
			MessagesParams = append(MessagesParams, openai.UserMessage(m.Content))
		case "assistant":
			MessagesParams = append(MessagesParams, openai.AssistantMessage(m.Content))
		}
	}
	params := openai.ChatCompletionNewParams{
		Model:           a.model,
		Messages:        MessagesParams,
		ReasoningEffort: shared.ReasoningEffortNone,
	}
	stream := a.client.Chat.Completions.NewStreaming(ctx, params)
	return stream

}

// Close 关闭客户端，释放资源
func (a *QwenAdapter) Close() error {
	return a.qwenClient.close()
}

// ChatWithToolsStream 工具感知的流式对话（Agent 核心能力）
// 在 Responses API 上挂载 Tools，流式解析文本增量与 function_call 事件，
// 转换为领域层 domain.AgentStreamEvent 流。
// 当 req.Tools 为空时，等价于纯文本流式对话（与 ChatHistoryStream 行为一致）。
func (a *QwenAdapter) ChatWithToolsStream(ctx context.Context, req domain.AgentRequest) *ssestream.Stream[domain.AgentStreamEvent] {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch {
		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			// assistant 发起的工具调用：转为 function_call 输入项，
			// 使下一轮 LLM 能看到"我上一轮调了哪些工具、参数是什么"。
			if strings.TrimSpace(m.Content) != "" {
				items = append(items, responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleAssistant,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openai.String(m.Content),
						},
					},
				})
			}
			for _, tc := range m.ToolCalls {
				items = append(items, responses.ResponseInputItemUnionParam{
					OfFunctionCall: &responses.ResponseFunctionToolCallParam{
						CallID:    tc.ID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
						Type:      constant.FunctionCall("function_call"),
					},
				})
			}
		case m.Role == "tool":
			// 工具执行结果：转为 function_call_output 输入项，CallID 关联上面的调用
			items = append(items, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					CallID: m.ToolCallID,
					Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
						OfString: param.NewOpt(m.Content),
					},
					Type: constant.FunctionCallOutput("function_call_output"),
				},
			})
		default:
			items = append(items, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRole(m.Role),
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(m.Content),
					},
				},
			})
		}
	}

	params := responses.ResponseNewParams{
		Model: a.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam(items),
		},
		Reasoning: shared.ReasoningParam{
			Summary: "concise",
			Effort:  shared.ReasoningEffortNone,
		},
	}

	if len(req.Tools) > 0 {
		tools := make([]responses.ToolUnionParam, 0, len(req.Tools))
		for _, spec := range req.Tools {
			var schema map[string]any
			if spec.Parameters != "" {
				// 参数 schema 解析失败时不阻塞对话，退化为空对象 schema
				if err := json.Unmarshal([]byte(spec.Parameters), &schema); err != nil {
					logger.Warn("[Qwen] invalid tool parameters schema, fallback to empty object",
						"tool", spec.Name, "error", err)
				}
			}
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, responses.ToolParamOfFunction(spec.Name, schema, false))
		}
		params.Tools = tools
	}

	params.SetExtraFields(map[string]any{
		"enable_thinking": false, // 关闭思考模式，降低延迟（语音对话敏感）
	})

	inner := a.client.Responses.NewStreaming(ctx, params)
	return ssestream.NewStream[domain.AgentStreamEvent](
		&agentStreamDecoder{inner: inner, toolCalls: make(map[string]*domain.ToolCall)},
		nil,
	)
}

// agentStreamDecoder 将 Responses API 的流式事件转换为 domain.AgentStreamEvent 流。
// 它实现 ssestream.Decoder 接口，使 ChatWithToolsStream 能直接返回 *ssestream.Stream[domain.AgentStreamEvent]。
// 转换策略：
//   - response.output_text.delta → 立即产出一条文本增量事件
//   - response.function_call_arguments.delta/done → 按 ItemID 累积参数，记录工具名
//   - 流结束时 → 产出一条 IsDone=true 的终态事件，携带本次累积的全部 ToolCalls（一次性给全）
type agentStreamDecoder struct {
	inner     *ssestream.Stream[responses.ResponseStreamEventUnion]
	toolCalls map[string]*domain.ToolCall
	order     []string
	cur       ssestream.Event
	finished  bool
	err       error
}

func (d *agentStreamDecoder) Event() ssestream.Event { return d.cur }
func (d *agentStreamDecoder) Err() error             { return d.err }
func (d *agentStreamDecoder) Close() error {
	if d.inner != nil {
		return d.inner.Close()
	}
	return nil
}

func (d *agentStreamDecoder) Next() bool {
	if d.finished {
		return false
	}
	for d.inner.Next() {
		ev := d.inner.Current()
		switch ev.Type {
		case "response.output_text.delta":
			delta := ev.AsResponseOutputTextDelta().Delta
			if delta != "" {
				d.cur = ssestream.Event{Data: mustJSONAgent(domain.AgentStreamEvent{Text: delta})}
				return true
			}
		case "response.function_call_arguments.delta":
			fc := ev.AsResponseFunctionCallArgumentsDelta()
			tc := d.toolCalls[fc.ItemID]
			if tc == nil {
				tc = &domain.ToolCall{ID: fc.ItemID}
				d.toolCalls[fc.ItemID] = tc
				d.order = append(d.order, fc.ItemID)
			}
			tc.Arguments += fc.Delta
		case "response.function_call_arguments.done":
			fc := ev.AsResponseFunctionCallArgumentsDone()
			tc := d.toolCalls[fc.ItemID]
			if tc == nil {
				tc = &domain.ToolCall{ID: fc.ItemID}
				d.toolCalls[fc.ItemID] = tc
				d.order = append(d.order, fc.ItemID)
			}
			tc.Name = fc.Name
			tc.Arguments = fc.Arguments
		}
	}

	// 内层流结束：把累积的工具调用作为终态事件一次性给出
	if d.inner.Err() != nil {
		d.err = d.inner.Err()
		return false
	}
	if len(d.order) > 0 {
		calls := make([]domain.ToolCall, 0, len(d.order))
		for _, id := range d.order {
			calls = append(calls, *d.toolCalls[id])
		}
		d.cur = ssestream.Event{Data: mustJSONAgent(domain.AgentStreamEvent{ToolCalls: calls, IsDone: true})}
		d.finished = true
		return true
	}
	// 本轮为纯文本回复（无工具调用）：仍需发出 IsDone:true 终态事件，
	// 否则上层的 reactLoopDecoder 无法判定单步结束，会 Infinite re-call LLM。
	d.cur = ssestream.Event{Data: mustJSONAgent(domain.AgentStreamEvent{IsDone: true})}
	d.finished = true
	return true
}

// mustJSONAgent 序列化 AgentStreamEvent；失败返回空对象，避免流式消费 panic。
func mustJSONAgent(v domain.AgentStreamEvent) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		logger.Warn("[Qwen] marshal AgentStreamEvent failed", "error", err)
		return []byte("{}")
	}
	return b
}
