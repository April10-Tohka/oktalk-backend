package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"pronunciation-correction-system/internal/domain"

	"github.com/openai/openai-go/v3/packages/ssestream"
)

// maxSteps ReAct 循环的最大步数（防护：避免 LLM 无限调工具造成延迟/成本失控）。
// 达到上限后，最后一步会强制以"纯文本"收尾，确保本轮一定给出自然语言回复。
const maxSteps = 4

// turnTimeout 单轮 LLM 调用的总超时（P4 打磨项）。
// 包裹每次 ChatWithToolsStream 的 context，避免某个 LLM 步骤因网络/限流无限挂起，
// 导致整轮对话卡死。超时后会由 decoder 的 Err() 路径安全结束本轮。
// 设为较宽松值（2 分钟），覆盖"多步工具 + 儿童短回复"的最坏延迟。
const turnTimeout = 2 * time.Minute

// Agent 语音对话 Agent 编排层。
//
// 设计定位：替换 free_talk.go 中 llmGoroutine 里"直接调 LLM"的逻辑，成为
// ASR 与 TTS 之间的"大脑 + 规划 + 执行"层。RunTurn 通过 ReAct 式多步循环
// 驱动 LLM：流式产出文本的同时，若 LLM 发起工具调用则执行并把结果回填，
// 循环至 LLM 产出"终态文本"（无新工具调用）为止。
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

// RunTurn 执行一轮对话（含潜在的多次工具调用 ReAct 循环）。
//
// 返回 *ssestream.Stream[domain.AgentStreamEvent]，对 free_talk.go 透明：
//   - 各步 LLM 产出的文本增量以 ev.Text 形式流出（直接喂 TTS，保证边想边说）；
//   - 终态以一条 IsDone=true 事件收尾（无 ToolCalls 字段，工具已在内部执行完毕）。
//
// ReAct 循环由 reactLoopDecoder 在流内部驱动：每一步调一次
// llm.ChatWithToolsStream，收集 ToolCalls → 执行 → 回填 messages → 继续，
// 直至某一步不再发起工具调用。这样 free_talk.go 无需感知"多步"的存在。
func (a *Agent) RunTurn(ctx context.Context, messages []domain.Message) *ssestream.Stream[domain.AgentStreamEvent] {
	return ssestream.NewStream[domain.AgentStreamEvent](
		&reactLoopDecoder{
			agent:    a,
			ctx:      ctx,
			messages: messages,
		},
		nil,
	)
}

// reactLoopDecoder 实现 ssestream.Decoder，在流内部驱动 ReAct 多步循环，
// 把多步 LLM 调用的文本流合并成一条对消费方透明的 AgentStreamEvent 流。
type reactLoopDecoder struct {
	agent    *Agent
	ctx      context.Context
	messages []domain.Message

	step       int
	stepStream *ssestream.Stream[domain.AgentStreamEvent]
	stepCancel context.CancelFunc // 当前步 LLM 调用的超时 context 取消函数
	cur        ssestream.Event
	finished   bool
	errored    bool
	err        error
}

func (d *reactLoopDecoder) Event() ssestream.Event { return d.cur }
func (d *reactLoopDecoder) Err() error             { return d.err }

// endStep 结束当前步：取消其超时 context 并清空步流，准备进入下一步或收尾。
func (d *reactLoopDecoder) endStep() {
	if d.stepCancel != nil {
		d.stepCancel()
		d.stepCancel = nil
	}
	d.stepStream = nil
}

func (d *reactLoopDecoder) Close() error {
	if d.stepStream != nil {
		_ = d.stepStream.Close()
	}
	if d.stepCancel != nil {
		d.stepCancel()
	}
	return nil
}

// Next 由 ssestream.Stream.Next 驱动。每次返回 true 表示 cur 中已准备好一条
// AgentStreamEvent（文本增量或终态），false 表示整个 ReAct 循环结束或出错。
func (d *reactLoopDecoder) Next() bool {
	if d.finished || d.errored {
		return false
	}
	for {
		// 1. 确保本步有可用的 LLM 流
		if d.stepStream == nil {
			// 达到步数上限：最后一步强制纯文本，避免工具循环无法收尾
			tools := d.agent.registry.Specs()
			if d.step >= maxSteps {
				tools = nil
			}
			// 单轮 LLM 超时兜底（P4）：包裹 context，避免本轮无限挂起
			stepCtx, stepCancel := context.WithTimeout(d.ctx, turnTimeout)
			d.stepCancel = stepCancel
			req := domain.AgentRequest{Messages: d.messages, Tools: tools}
			d.stepStream = d.agent.llm.ChatWithToolsStream(stepCtx, req)
		}

		// 2. 从本步流中读取下一个事件
		if !d.stepStream.Next() {
			if d.stepStream.Err() != nil {
				d.err = d.stepStream.Err()
				d.errored = true
				return false
			}
			// 本步流结束（理论不应发生，adapter 必发一条 IsDone 终态）
			d.endStep()
			continue
		}

		ev := d.stepStream.Current()

		// 3. 终态事件：判断是否还需要执行工具
		if ev.IsDone {
			d.endStep()
			if len(ev.ToolCalls) == 0 {
				// 无工具调用 → 本轮 ReAct 循环结束，发出干净终态
				d.cur = ssestream.Event{Data: mustJSONAgent(domain.AgentStreamEvent{IsDone: true})}
				d.finished = true
				return true
			}
			// 有工具调用 → 执行并把结果作为 observation 回填，继续循环
			d.executeStep(ev.ToolCalls)
			d.step++
			continue
		}

		// 4. 普通文本增量 → 透传给 TTS（边想边说）
		if ev.Text != "" {
			d.cur = ssestream.Event{Data: mustJSONAgent(domain.AgentStreamEvent{Text: ev.Text})}
			return true
		}
		// 空文本非终态事件（如纯工具步无过渡句），跳过
		continue
	}
}

// executeStep 记录本步 assistant 发起的工具调用，并逐个执行、将结果作为
// tool 角色消息回填，供下一轮 LLM 看到"我调了什么、得到了什么"。
func (d *reactLoopDecoder) executeStep(calls []domain.ToolCall) {
	d.messages = append(d.messages, domain.Message{Role: "assistant", ToolCalls: calls})
	for _, call := range calls {
		result := d.agent.registry.Execute(d.ctx, call)
		d.messages = append(d.messages, domain.Message{Role: "tool", ToolCallID: call.ID, Content: result})
	}
}

// mustJSONAgent 序列化 AgentStreamEvent；失败返回空对象，避免流式消费 panic。
func mustJSONAgent(v domain.AgentStreamEvent) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Warn("[Agent] marshal AgentStreamEvent failed", "error", err)
		return []byte("{}")
	}
	return b
}
