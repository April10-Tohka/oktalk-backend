// Package domain 定义核心业务接口
// 所有接口方法只使用 Go 原生类型，严禁出现任何第三方 SDK 结构体
package domain

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

type Message struct {
	Role string // "system" | "user" | "assistant" | "tool"
	// Content 文本内容（system/user/assistant/tool 通用）
	Content string
	// ToolCalls 当本消息是 assistant 且发起了工具调用时，记录本次调用列表
	// 用于 ReAct 循环把"我调了哪些工具"回填给下一轮 LLM
	ToolCalls []ToolCall
	// ToolCallID 当 Role=="tool" 时，标记本结果是哪个工具调用（ToolCall.ID）的 observation
	ToolCallID string
}

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

	ChatHistoryStream(ctx context.Context, messages []Message) *ssestream.Stream[openai.ChatCompletionChunk]

	// ChatWithToolsStream 工具感知的流式对话（Agent 核心契约）
	// 与 ChatHistoryStream 类似，但允许 LLM 在流式产出文本的同时发起工具调用。
	// 返回的流中：文本增量以 AgentStreamEvent.Text 形式到达；
	// 工具调用在终态事件（IsDone=true）中一并返回（通常一次性给全）。
	// 若 req.Tools 为空，则等价于纯文本流式对话，行为与 ChatHistoryStream 一致。
	ChatWithToolsStream(ctx context.Context, req AgentRequest) *ssestream.Stream[AgentStreamEvent]

	// Close 关闭客户端，释放资源
	Close() error
}

// ===================== Agent 工具调用契约（P0） =====================

// ToolSpec 工具的描述信息，用于构建给 LLM 的 ToolSpec（让 LLM 知道何时该调哪个工具）。
// Description 应写成"龙宝 OK 视角"的指令，便于模型决策。
type ToolSpec struct {
	Name        string // 工具名，如 "assess_pronunciation"
	Description string // 给 LLM 的自然语言说明
	Parameters  string // JSON Schema 字符串，描述参数结构（"type":"object" + "properties"）
}

// ToolCall LLM 产生的工具调用请求（Agent 解析后交给 Registry 执行）。
type ToolCall struct {
	ID        string `json:"id"`        // 工具调用唯一 id，回灌 observation 时使用
	Name      string `json:"name"`      // 工具名
	Arguments string `json:"arguments"` // JSON 字符串，工具参数
}

// AgentStreamEvent Agent 流式产出事件。
// 一次 LLM 流式响应会被转换为一连串 AgentStreamEvent：
//   - 文本增量：Text 非空、IsDone=false
//   - 终态事件：IsDone=true，可能携带本步产生的 ToolCalls（一次性给全）
type AgentStreamEvent struct {
	Text      string     `json:"text"`       // 增量文本（可能为空）
	IsDone    bool       `json:"is_done"`    // 本轮（含工具循环）是否结束
	ToolCalls []ToolCall `json:"tool_calls"` // 本步产生的工具调用（非空即代表需要执行）
}

// AgentRequest 工具感知的对话请求。
type AgentRequest struct {
	Messages []Message  // 完整对话历史（含 system）
	Tools    []ToolSpec // 当前开放给 LLM 的工具集合（空=纯对话）
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
