package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"pronunciation-correction-system/internal/domain"
)

// ===================== 常量配置 =====================

const (
	// maxTurns 滑动窗口最大轮数（一轮 = 用户说一次 + AI 回一次）
	// 超过此值时触发对话重建，将历史压缩为摘要注入新 System Prompt
	maxTurns = 6

	// rebuildThreshold 触发重建的轮数阈值，略小于 maxTurns
	// 在第 rebuildThreshold 轮结束后就预先重建，避免最后一轮才重建带来的延迟
	rebuildThreshold = 5
)

// ===================== 数据结构 =====================

// Turn 单轮对话记录（用户一句 + AI 一句）
type Turn struct {
	UserText      string // ASR 识别结果（用户说的英文）
	AssistantText string // LLM 完整回复
}

// ConversationMemory 短期对话记忆管理器
//
// 职责：
//  1. 维护本地轻量历史（[]Turn），用于摘要生成和上下文感知
//  2. 持有当前 conversationID，对话实际由服务端维护
//  3. 当轮数到达 rebuildThreshold 时，生成历史摘要并重建新对话
//     重建对新 conversationID 透明，llmGoroutine 无需感知
//
// 线程安全说明：
//
//	ConversationMemory 设计为单 goroutine 使用（仅 llmGoroutine 读写）
//	不需要加锁
type ConversationMemory struct {
	// history 已完成的对话轮次（滑动窗口）
	history []Turn

	// pendingUserText 当前轮次用户的输入，等待 AI 回复后配对存入 history
	pendingUserText string

	// currentAssistant 当前轮次 AI 回复的累积文本（流式 token 逐步追加）
	currentAssistant strings.Builder

	// convID 当前有效的服务端对话 ID
	convID string

	// llmProvider 用于重建对话时调用 NewConversation
	llmProvider domain.LLMProvider

	// userProfile 用户画像（名字、年龄等），用于组装 System Prompt
	userProfile UserProfile

	// turnCount 已完成的总轮次数（含重建前的历史，用于日志）
	turnCount int
}

// UserProfile 用户画像（P0 阶段保持简单，后续 P2 扩展）
type UserProfile struct {
	// Name 孩子的名字，AI 对话时会用到
	Name string

	// AgeGroup 年龄分组，影响词汇难度和话题选择
	// 取值："6-8" | "9-12"
	AgeGroup string

	// PreferredTopics 兴趣话题（可为空，为空时 AI 自由发挥）
	PreferredTopics []string
}

// ===================== 构造函数 =====================

// NewConversationMemory 初始化记忆管理器并创建第一个服务端对话
//
// 参数：
//   - ctx: 请求上下文
//   - llmProvider: LLM 接口，用于 NewConversation
//   - profile: 用户画像
//
// 返回已初始化的 ConversationMemory，首次 System Prompt 已注入
func NewConversationMemory(ctx context.Context, llmProvider domain.LLMProvider, profile UserProfile) (*ConversationMemory, error) {
	m := &ConversationMemory{
		llmProvider: llmProvider,
		userProfile: profile,
	}

	systemPrompt := m.buildSystemPrompt("")
	convID, err := llmProvider.NewConversation(ctx, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("conversation memory init failed: %w", err)
	}

	m.convID = convID
	slog.Info("[Memory] Conversation created",
		"conv_id", convID,
		"age_group", profile.AgeGroup,
	)
	return m, nil
}

// ===================== 核心方法 =====================

// ConvID 返回当前有效的服务端对话 ID，供 llmGoroutine 调用 ConversationChatStream
func (m *ConversationMemory) ConvID() string {
	return m.convID
}

// OnUserInput 在 ASR 返回文本后调用，记录本轮用户输入
// 必须在调用 LLM 之前调用
func (m *ConversationMemory) OnUserInput(text string) {
	m.pendingUserText = text
	m.currentAssistant.Reset()
}

// OnAssistantToken 每收到一个流式 token 调用一次，追加到当前轮次的 AI 回复
func (m *ConversationMemory) OnAssistantToken(token string) {
	m.currentAssistant.WriteString(token)
}

// OnTurnComplete 在 LLM 本轮输出结束后调用（即收到 IsDone:true 后）
//
// 将本轮 (user, assistant) 配对存入 history，并检查是否需要重建对话。
// 重建是异步发起的（在下一轮输入到来之前完成），不阻塞当前回复的流式输出。
//
// 参数：
//   - ctx: 用于重建对话时调用 NewConversation
func (m *ConversationMemory) OnTurnComplete(ctx context.Context) {
	turn := Turn{
		UserText:      m.pendingUserText,
		AssistantText: m.currentAssistant.String(),
	}
	m.history = append(m.history, turn)
	m.turnCount++
	m.pendingUserText = ""
	m.currentAssistant.Reset()

	slog.Debug("[Memory] Turn recorded",
		"turn_count", m.turnCount,
		"history_len", len(m.history),
		"user_preview", preview(turn.UserText, 30),
	)

	// 达到重建阈值：压缩历史，重建新的服务端对话
	// 在下一轮用户说话前（有一定静默时间）完成，几乎无感知
	if len(m.history) >= rebuildThreshold {
		m.rebuild(ctx)
	}
}

// ===================== 内部方法 =====================

// rebuild 用现有历史生成摘要，重建新的服务端对话
//
// 重建后 m.convID 更新为新 ID，m.history 重置为空（摘要已编码进 System Prompt）
// 如果重建失败，保留旧 convID 继续使用，记录 warn 日志但不中断会话
func (m *ConversationMemory) rebuild(ctx context.Context) {
	summary := m.buildHistorySummary()
	newSystemPrompt := m.buildSystemPrompt(summary)

	newConvID, err := m.llmProvider.NewConversation(ctx, newSystemPrompt)
	if err != nil {
		// 重建失败不致命：旧对话仍可用，只是 token 会越来越长
		slog.Warn("[Memory] Conversation rebuild failed, keeping old conv_id",
			"error", err,
			"old_conv_id", m.convID,
		)
		return
	}

	slog.Info("[Memory] Conversation rebuilt",
		"old_conv_id", m.convID,
		"new_conv_id", newConvID,
		"summarized_turns", len(m.history),
	)

	m.convID = newConvID
	// 清空已被摘要化的历史，从零开始积累新窗口
	m.history = m.history[:0]
}

// buildSystemPrompt 组装完整的 System Prompt
//
// summary 为空时生成首次 System Prompt（无历史）
// summary 非空时将历史摘要嵌入，给新对话提供上下文延续感
func (m *ConversationMemory) buildSystemPrompt(summary string) string {
	p := m.userProfile
	var sb strings.Builder

	// ── 角色定义 ──
	sb.WriteString("You are Mia, a warm and encouraging English speaking coach for Chinese children.\n")
	sb.WriteString("You are talking with ")
	if p.Name != "" {
		sb.WriteString(p.Name)
	} else {
		sb.WriteString("a student")
	}
	sb.WriteString(".\n\n")

	// ── 年龄适配 ──
	sb.WriteString("## Student profile\n")
	switch p.AgeGroup {
	case "6-8":
		sb.WriteString("- Age: 6–8 years old. Use very simple words (A1–A2 level). Keep sentences short (under 10 words). Be playful and use lots of encouragement.\n")
	case "9-12":
		sb.WriteString("- Age: 9–12 years old. Use everyday vocabulary (A2–B1 level). You can ask follow-up questions and introduce new words naturally.\n")
	default:
		sb.WriteString("- Age: primary school child. Adjust difficulty based on responses. Start simple.\n")
	}

	if len(p.PreferredTopics) > 0 {
		sb.WriteString("- Favorite topics: ")
		sb.WriteString(strings.Join(p.PreferredTopics, ", "))
		sb.WriteString(".\n")
	}

	// ── 教学原则 ──
	sb.WriteString("\n## Teaching principles\n")
	sb.WriteString("- Always respond in English only.\n")
	sb.WriteString("- Keep each reply concise: 1–3 sentences maximum.\n")
	sb.WriteString("- If the child makes a grammar mistake, use implicit recast (repeat correctly without pointing out the error).\n")
	sb.WriteString("  Example: Child says \"I goed to park\", you say \"Oh, you went to the park! That sounds fun!\"\n")
	sb.WriteString("- If the child goes silent or gives a very short answer, offer a simple choice to help them continue:\n")
	sb.WriteString("  Example: \"Do you like cats or dogs?\"\n")
	sb.WriteString("- End each reply with one simple open question to keep the conversation going.\n")
	sb.WriteString("- Never correct explicitly. Never say 'wrong' or 'mistake'.\n")
	sb.WriteString("- Use encouraging phrases naturally: 'Great!', 'Wow!', 'That's interesting!'\n")

	// ── 历史摘要（重建时注入）──
	if summary != "" {
		sb.WriteString("\n## Conversation so far\n")
		sb.WriteString("You have already been talking for a while. Here is a brief summary of what was discussed:\n")
		sb.WriteString(summary)
		sb.WriteString("\nContinue the conversation naturally as if nothing changed.\n")
	}

	return sb.String()
}

// buildHistorySummary 将 m.history 压缩为自然语言摘要
//
// 摘要格式设计原则：
//   - 保留话题和关键词，让 AI 能延续话题
//   - 不超过 300 字，避免占用过多 context window
//   - 用第三人称描述，适合作为 System Prompt 的一部分
func (m *ConversationMemory) buildHistorySummary() string {
	if len(m.history) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Topics discussed and key points:\n")

	for i, t := range m.history {
		sb.WriteString(fmt.Sprintf("- Turn %d: Student said: \"%s\" | You replied: \"%s\"\n",
			i+1,
			preview(t.UserText, 60),
			preview(t.AssistantText, 80),
		))
	}

	return sb.String()
}

// preview 截取字符串前 n 个字符，超出时加省略号
// 用于日志和摘要，避免过长
func preview(s string, n int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
