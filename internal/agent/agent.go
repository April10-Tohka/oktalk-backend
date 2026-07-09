package agent

import (
	"context"

	"pronunciation-correction-system/internal/domain"

	"github.com/openai/openai-go/v3/packages/ssestream"
)

// maxSteps ReAct 循环的最大步数（防护：避免 LLM 无限调工具造成延迟/成本失控）。
const maxSteps = 4

// Agent 语音对话 Agent 编排层（P0 骨架）。
//
// 设计定位：替换 free_talk.go 中 llmGoroutine 里"直接调 LLM"的逻辑，成为
// ASR 与 TTS 之间的"大脑 + 规划 + 执行"层。P0 阶段 RunTurn 为透传模式：
// 直接委托 LLMProvider.ChatWithToolsStream 流式产出文本，不发起任何工具调用
// （注册表为空），因此对话行为与改造前完全一致。
type Agent struct {
	llm      domain.LLMProvider
	registry *Registry
}

// NewAgent 创建 Agent。registry 为 nil 时自动创建空注册表（等价于纯对话透传）。
func NewAgent(llm domain.LLMProvider, registry *Registry) *Agent {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Agent{
		llm:      llm,
		registry: registry,
	}
}

// Registry 返回当前工具注册表，供调用方在后续阶段按需注册工具。
func (a *Agent) Registry() *Registry {
	return a.registry
}

// RunTurn 执行一轮对话（含潜在的未来多步工具循环）。
//
// P0 透传实现：把 messages 与"当前开放的工具集合"交给 LLM 流式产出，返回文本事件流。
// 返回的 *ssestream.Stream[domain.AgentStreamEvent] 与 free_talk.go 既有的消费方式兼容
// （逐条读取 ev.Text 即可），其余 goroutine（VAD/ASR/TTS/句子切分）无需改动。
//
// 后续阶段（P1）将在此方法内叠加 ReAct 循环：
//
//	for step := 0; step < maxSteps; step++ {
//	    stream := a.llm.ChatWithToolsStream(ctx, req)
//	    // 流式转发 Text 给 TTS；收集本轮 ToolCalls
//	    // 若本步无 ToolCalls → break（终态）
//	    // 否则逐个执行 a.registry.Execute，把结果作为 observation 追加进 messages，继续循环
//	}
func (a *Agent) RunTurn(ctx context.Context, messages []domain.Message) *ssestream.Stream[domain.AgentStreamEvent] {
	req := domain.AgentRequest{
		Messages: messages,
		Tools:    a.registry.Specs(),
	}
	return a.llm.ChatWithToolsStream(ctx, req)
}
