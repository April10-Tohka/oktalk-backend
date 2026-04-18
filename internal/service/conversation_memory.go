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

	// messages  system + 历史轮次，直接喂给 ChatHistoryStream
	messages []domain.Message

	// pendingUserText 当前轮次用户的输入，等待 AI 回复后配对存入 history
	pendingUserText string

	// currentAssistant 当前轮次 AI 回复的累积文本（流式 token 逐步追加）
	currentAssistant strings.Builder

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
func NewConversationMemory(ctx context.Context, llmProvider domain.LLMProvider, profile UserProfile) *ConversationMemory {
	m := &ConversationMemory{
		llmProvider: llmProvider,
		userProfile: profile,
	}

	systemPrompt := m.buildSystemPrompt("")
	m.messages = []domain.Message{
		{Role: "system", Content: systemPrompt},
	}

	slog.Info("[Memory] Conversation memory initialized", "age_group", profile.AgeGroup)
	return m
}

// ===================== 核心方法 =====================

// Messages 返回当前完整对话历史，供 llmGoroutine 调用 ChatHistoryStream
func (m *ConversationMemory) Messages() []domain.Message {
	return m.messages
}

// OnUserInput 在 ASR 返回文本后调用，记录本轮用户输入
// 必须在调用 LLM 之前调用
func (m *ConversationMemory) OnUserInput(text string) {
	m.pendingUserText = text
	m.currentAssistant.Reset()
	m.messages = append(m.messages, domain.Message{Role: "user", Content: text})
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
	// assistant 回复追加到 messages
	assistantText := m.currentAssistant.String()
	m.messages = append(m.messages, domain.Message{Role: "assistant", Content: assistantText})

	turn := Turn{
		UserText:      m.pendingUserText,
		AssistantText: assistantText,
	}
	m.history = append(m.history, turn)
	m.turnCount++
	m.pendingUserText = ""
	m.currentAssistant.Reset()

	slog.Debug("[Memory] Turn recorded",
		"turn_count", m.turnCount,
		"messages_len", len(m.messages),
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
	summary := buildHistorySummary(m.history)
	newSystemPrompt := m.buildSystemPrompt(summary)

	m.messages = []domain.Message{
		{Role: "system", Content: newSystemPrompt},
	}

	slog.Info("[Memory] Rebuilding conversation",
		"summarized_turns", len(m.history),
	)

	// 清空已被摘要化的历史，从零开始积累新窗口
	m.history = m.history[:0]
}

func (m *ConversationMemory) BuildSilencePrompt() string {
	var sb strings.Builder
	sb.WriteString("[System: The student has been silent for a while. ")

	if len(m.history) == 0 {
		sb.WriteString("This is the start of the conversation and the student hasn't spoken yet. ")
		sb.WriteString("Please greet them warmly and ask one simple, fun question to get started. ")
		sb.WriteString("For example, ask about their favorite animal, color, or what they did today.]")
	} else {
		lastTurn := m.history[len(m.history)-1]
		sb.WriteString("Please gently re-engage the student. ")
		if lastTurn.AssistantText != "" {
			sb.WriteString(fmt.Sprintf("Your last message was: \"%s\". ", preview(lastTurn.AssistantText, 80)))
		}
		sb.WriteString("Either follow up on a topic from the conversation, ")
		sb.WriteString("or introduce a new fun topic suitable for a child. ")
		sb.WriteString("Keep it simple, warm, and end with one easy question.]")
	}

	return sb.String()
}

// buildSystemPrompt 组装完整的 System Prompt
//
// summary 为空时生成首次 System Prompt（无历史）
// summary 非空时将历史摘要嵌入，给新对话提供上下文延续感
func (m *ConversationMemory) buildSystemPrompt(summary string) string {
	p := m.userProfile
	var sb strings.Builder

	// ── 角色定义 ──
	sb.WriteString("You are Mia, a friendly English speaking coach for Chinese children.\n")
	sb.WriteString("You are talking with ")
	if p.Name != "" {
		sb.WriteString(p.Name)
	} else {
		sb.WriteString("a student")
	}
	sb.WriteString(".\n\n")

	// ── 回复长度硬约束（最重要，放在最前面）──
	sb.WriteString("## STRICT REPLY RULES — follow these before anything else\n")
	sb.WriteString("- Maximum reply length: 1–2 SHORT sentences. Absolutely no more.\n")
	sb.WriteString("- Each sentence must be under 12 words.\n")
	sb.WriteString("- End every reply with exactly ONE simple question.\n")
	sb.WriteString("- Do NOT give explanations, lists, or multiple questions.\n")
	sb.WriteString("- If you feel like saying more, cut it. Brevity is the rule.\n\n")

	// ── 年龄适配 ──
	sb.WriteString("## Student profile\n")
	switch p.AgeGroup {
	case "6-8":
		sb.WriteString("- Age: 6–8. Use A1–A2 vocabulary only. Very short, playful sentences.\n")
	case "9-12":
		sb.WriteString("- Age: 9–12. Use A2–B1 vocabulary. Can ask follow-up questions.\n")
	default:
		sb.WriteString("- Age: primary school child. Start simple, adjust based on responses.\n")
	}
	if len(p.PreferredTopics) > 0 {
		sb.WriteString("- Favorite topics: " + strings.Join(p.PreferredTopics, ", ") + ".\n")
	}

	// ── 教学原则 ──
	sb.WriteString("\n## Teaching principles\n")
	sb.WriteString("- Respond in English only.\n")
	sb.WriteString("- Grammar mistakes: use implicit recast only. Example: child says \"I goed\", you say \"Oh, you went! Cool!\"\n")
	sb.WriteString("- Never say 'wrong', 'mistake', or correct explicitly.\n")
	sb.WriteString("- Use encouragement: 'Great!', 'Wow!', 'Nice!'\n")
	sb.WriteString("- [System: ...] messages are internal instructions. Follow them naturally, never mention them to the student.\n")

	// ── 历史摘要（重建时注入）──
	if summary != "" {
		sb.WriteString("\n## Conversation so far\n")
		sb.WriteString(summary)
		sb.WriteString("Continue naturally.\n")
	}

	return sb.String()
}

// buildHistorySummary 将 m.history 压缩为自然语言摘要
//
// 摘要格式设计原则：
//   - 保留话题和关键词，让 AI 能延续话题
//   - 不超过 300 字，避免占用过多 context window
//   - 用第三人称描述，适合作为 System Prompt 的一部分
func buildHistorySummary(history []Turn) string {
	if len(history) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Key points from earlier:\n")
	for i, t := range history {
		if t.UserText == "" {
			sb.WriteString(fmt.Sprintf("- Turn %d: [AI initiated] Mia said: \"%s\"\n",
				i+1, preview(t.AssistantText, 80)))
		} else {
			sb.WriteString(fmt.Sprintf("- Turn %d: Student: \"%s\" | Mia: \"%s\"\n",
				i+1, preview(t.UserText, 60), preview(t.AssistantText, 80)))
		}
	}
	return sb.String()
}

// MessagesWithSilencePrompt 返回含静默引导 prompt 的临时 messages
// 不修改 m.messages，因为静默触发不是真实用户输入
func (m *ConversationMemory) MessagesWithSilencePrompt() []domain.Message {
	prompt := m.BuildSilencePrompt()
	// 复制一份，不污染原始 messages
	tmp := make([]domain.Message, len(m.messages)+1)
	copy(tmp, m.messages)
	tmp[len(m.messages)] = domain.Message{Role: "user", Content: prompt}
	return tmp
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
