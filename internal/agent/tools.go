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
	"log/slog"
	"sync"
	"time"

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
//
// 安全与稳定性防护（P3/P4 打磨项）：
//   - allowList：儿童安全白名单。若非 nil，只有名单内的工具名可被调用；
//     即使 LLM 伪造了某个已注册工具名之外的调用也会被拒绝（纵深防御）。
//   - budget / used：跨会话（整个 Session）的工具调用总预算。达到上限后，
//     后续调用返回"预算耗尽"安全提示，强制本轮以自然语言收尾，防止失控刷工具。
//   - toolTimeout：单次工具执行的超时上限。包裹 Execute 的 context，避免某个
//     慢/挂起的工具（如外部服务抖动）无限阻塞整个 ReAct 循环。
type Registry struct {
	tools map[string]Tool

	// allowList 非 nil 时作为儿童安全白名单；只有其中工具名允许执行。
	allowList map[string]bool

	budgetMu sync.Mutex
	budget   int // 0 = 不限；>0 = 整个 Session 允许的最大工具调用次数
	used     int // 已消耗次数

	toolTimeout time.Duration // 单次工具执行超时；0 = 不限制
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

// AllowTools 设定儿童安全白名单。只有 names 中的工具名可被调用；
// 传空切片则清空白名单（退化为"仅已注册工具可调用"）。
// 注意：白名单仅作纵深防御，调用入口本就只允许已注册工具。
func (r *Registry) AllowTools(names ...string) {
	if len(names) == 0 {
		r.allowList = nil
		return
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	r.allowList = m
}

// SetToolTimeout 设定单次工具执行的超时上限。d<=0 表示不限制。
func (r *Registry) SetToolTimeout(d time.Duration) {
	r.toolTimeout = d
}

// SetBudget 设定整个 Session 的工具调用总预算。n<=0 表示不限。
func (r *Registry) SetBudget(n int) {
	r.budgetMu.Lock()
	defer r.budgetMu.Unlock()
	r.budget = n
}

// ToolCallsUsed 返回已消耗的工具调用次数（用于观察/测试）。
func (r *Registry) ToolCallsUsed() int {
	r.budgetMu.Lock()
	defer r.budgetMu.Unlock()
	return r.used
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
//
// 安全/稳定性防护顺序：
//  1. 白名单拦截（儿童安全）：非白名单工具名直接拒绝并记录告警；
//  2. 预算拦截：跨会话总调用数达上限后，返回"预算耗尽"提示，强制本轮收尾；
//  3. 超时包裹：单次执行受 toolTimeout 约束，避免慢工具阻塞循环。
func (r *Registry) Execute(ctx context.Context, call domain.ToolCall) string {
	// 1. 儿童安全白名单（纵深防御）：拒绝名单外的工具名
	if r.allowList != nil && !r.allowList[call.Name] {
		slog.Warn("[Agent] tool call rejected by allowlist (child-safety)",
			"tool", call.Name)
		return "tool not allowed: " + call.Name
	}

	t, ok := r.tools[call.Name]
	if !ok {
		return "tool not found: " + call.Name
	}

	// 2. 跨会话工具调用预算
	r.budgetMu.Lock()
	if r.budget > 0 && r.used >= r.budget {
		r.budgetMu.Unlock()
		slog.Warn("[Agent] tool call budget exhausted",
			"used", r.used, "budget", r.budget)
		return "tool call budget exhausted; please finish this turn with a short, natural reply to the child."
	}
	r.used++
	r.budgetMu.Unlock()

	// 3. 单次执行超时包裹
	execCtx := ctx
	var cancel context.CancelFunc
	if r.toolTimeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, r.toolTimeout)
		defer cancel()
	}

	res, err := t.Execute(execCtx, call.Arguments)
	if err != nil {
		return "tool error: " + err.Error()
	}
	return res
}
