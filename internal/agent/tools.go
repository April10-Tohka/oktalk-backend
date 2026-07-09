// Package agent 实现 AI 语音对话的 Agent 编排层（四要素：LLM + Planning + Tool use + Memory）。
//
// P0 阶段：本包为"骨架 + 透传"——定义 Tool/Registry 抽象与 Agent 结构体，
// RunTurn 仅把对话与"当前开放的工具"交给 LLMProvider 流式产出文本，不发起任何工具调用
// （注册表为空），行为与改造前的 ChatHistoryStream 完全一致，用于验证不破坏现有对话。
//
// 后续阶段将在此基础上叠加：
//   - Tool use：把已有 service（发音评测/练习单元/报告/用户画像）注册为具体 Tool；
//   - Planning：ReAct 式多步循环（runReactLoop）；
//   - Memory：会话级目标管理与长期/情景记忆。
package agent

import (
	"context"

	"pronunciation-correction-system/internal/domain"
)

// Tool Agent 可调用的工具抽象。
// Execute 返回可被 LLM 理解的文本结果（语音 Agent 不把原始音频直接塞给 LLM）。
type Tool interface {
	// Spec 返回工具的元信息，用于构建给 LLM 的 domain.ToolSpec。
	Spec() domain.ToolSpec
	// Execute 执行工具；args 为 LLM 传入的 JSON 字符串，返回文本结果（或错误）。
	Execute(ctx context.Context, args string) (string, error)
}

// Registry 工具注册表，按名称索引已注册工具。
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具（同名覆盖）。nil 会被忽略。
func (r *Registry) Register(t Tool) {
	if t == nil {
		return
	}
	r.tools[t.Spec().Name] = t
}

// Get 按名称取工具。
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Specs 返回所有已注册工具的元信息，用于构建 LLM 请求中的 Tools 列表。
func (r *Registry) Specs() []domain.ToolSpec {
	specs := make([]domain.ToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		specs = append(specs, t.Spec())
	}
	return specs
}

// Execute 执行一次工具调用，把结果或错误封装为 LLM 可消费的文本。
// 这样即使工具缺失或报错，Agent Loop 也不会中断（错误作为 observation 回灌 LLM）。
func (r *Registry) Execute(ctx context.Context, call domain.ToolCall) string {
	t, ok := r.tools[call.Name]
	if !ok {
		return "tool not found: " + call.Name
	}
	res, err := t.Execute(ctx, call.Arguments)
	if err != nil {
		return "tool error: " + err.Error()
	}
	return res
}
