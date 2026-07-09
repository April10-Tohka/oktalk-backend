package service

import (
	"context"
	"encoding/json"
	"fmt"

	agentpkg "pronunciation-correction-system/internal/agent"
	"pronunciation-correction-system/internal/domain"
)

// ===================== 同步工具实现（P1） =====================
//
// 这些工具实现 agentpkg.Tool，并在 Session.Run 中按"依赖非空"原则注册进
// Agent 的工具注册表。所有工具都返回可被 LLM 消费的精炼文本（JSON），
// 错误信息也被封装为文本回灌，避免 Agent Loop 因单个工具失败而中断。
//
// 注：工具实现放在 service 包（而非 internal/agent 包），是为了直接复用
// PronunciationService / ReportService 等能力，同时避免 internal/agent 反向
// 依赖 internal/service 造成的循环引用。

// ---------- 工具 1：list_pronunciation_units ----------

type listUnitsTool struct {
	svc *PronunciationService
}

func (t *listUnitsTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "list_pronunciation_units",
		Description: "查看所有可以练习的发音单元（主题/单词包）。当孩子或家长想开始一段发音练习、" +
			"或你想推荐练习内容时使用。参数 type 可选：'word'（单词）、'sentence'（句子），不填则返回全部。",
		Parameters: `{"type":"object","properties":{"type":{"type":"string","description":"单元类型：word 或 sentence，可留空返回全部"}}}`,
	}
}

func (t *listUnitsTool) Execute(ctx context.Context, args string) (string, error) {
	var p struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	list := t.svc.GetUnitList(ctx, p.Type)
	if len(list) == 0 {
		return "当前没有可用的发音练习单元。", nil
	}
	b, err := json.Marshal(list)
	if err != nil {
		return "查询到练习单元，但序列化失败。", nil
	}
	return string(b), nil
}

// ---------- 工具 2：get_user_profile ----------

type getUserProfileTool struct {
	profile UserProfile
}

func (t *getUserProfileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "get_user_profile",
		Description: "获取当前正在对话的小朋友的用户画像（名字、年龄段、感兴趣的话题）。" +
			"当需要称呼孩子的名字、选择适合难度的内容、或聊他感兴趣的话题时使用。",
		Parameters: `{"type":"object","properties":{}}`,
	}
}

func (t *getUserProfileTool) Execute(ctx context.Context, args string) (string, error) {
	b, err := json.Marshal(t.profile)
	if err != nil {
		return "获取用户画像失败。", nil
	}
	return string(b), nil
}

// ---------- 工具 3：get_learning_report ----------

type getLearningReportTool struct {
	svc    ReportService
	userID string
}

func (t *getLearningReportTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "get_learning_report",
		Description: "获取孩子最近一段时间（默认本周）的学习报告摘要：练习活跃度、发音能力、进步趋势、" +
			"高频难词、给孩子的鼓励。当孩子问'我最近学得怎么样'、'我哪些地方还要加油'时使用。",
		Parameters: `{"type":"object","properties":{"range":{"type":"string","description":"时间范围：weekly 或 monthly，默认 weekly"}}}`,
	}
}

func (t *getLearningReportTool) Execute(ctx context.Context, args string) (string, error) {
	if t.svc == nil {
		return "学习报告服务不可用。", nil
	}
	if t.userID == "" {
		return "未识别到当前用户，无法生成学习报告。", nil
	}
	var p struct {
		Range string `json:"range"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	if p.Range == "" {
		p.Range = "weekly"
	}
	resp, err := t.svc.ReportMVP(ctx, &ReportMVPRequest{ReportType: p.Range, UserID: t.userID})
	if err != nil {
		return fmt.Sprintf("获取学习报告失败：%s", err.Error()), nil
	}
	// 精简输出，只保留对口语对话有价值的字段，避免占用过多 token
	summary := map[string]interface{}{
		"report_type": resp.ReportType,
		"period":      map[string]string{"start": resp.PeriodStartDate, "end": resp.PeriodEndDate},
	}
	if resp.ActivityStats != nil {
		summary["activity"] = resp.ActivityStats
	}
	if resp.AbilityRadar != nil {
		summary["ability"] = resp.AbilityRadar
	}
	if resp.ProgressStats != nil {
		summary["progress"] = resp.ProgressStats
	}
	if resp.KidFriendlyCard != nil {
		summary["kid_card"] = resp.KidFriendlyCard
	}
	if resp.DifficultWords != nil {
		summary["difficult_words"] = resp.DifficultWords
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return "生成学习报告摘要失败。", nil
	}
	return string(b), nil
}

// registerAgentTools 按"依赖非空"原则，把 P1 的同步工具注册进 Agent 注册表。
// 任何 service 为 nil 时跳过对应工具，保证即便未注入依赖也不会 panic，
// 且此时注册表为空 → ReAct 循环退化为纯对话（与 P0 行为一致）。
func (s *Session) registerAgentTools(registry *agentpkg.Registry) {
	if s.pronunciationService != nil {
		registry.Register(&listUnitsTool{svc: s.pronunciationService})
	}
	registry.Register(&getUserProfileTool{profile: s.buildUserProfile()})
	if s.reportService != nil {
		registry.Register(&getLearningReportTool{svc: s.reportService, userID: s.userID})
	}
}
