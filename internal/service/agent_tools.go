package service

import (
	"context"
	"encoding/json"
	"fmt"

	agentpkg "pronunciation-correction-system/internal/agent"
	"pronunciation-correction-system/internal/domain"
)

// ===================== 同步 / 异步工具实现（P1 + P2） =====================
//
// 工具实现放在 service 包（而非 internal/agent 包），以复用 PronunciationService /
// ReportService / EvaluationProvider 等能力，同时避免 internal/agent 反向依赖
// internal/service 造成循环引用。
//
// 工具按"依赖非空"原则注册进 Agent 注册表：
//   - 任何 service 为 nil 时跳过对应工具，注册表退化为可用子集，保证不 panic；
//   - 未注入 evalProvider / pronunciationService 时，异步交互工具不出现，Agent 退化为纯对话+同步工具。

// ---------- 工具 1：list_pronunciation_units（同步） ----------

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

// ---------- 工具 2：get_user_profile（长期语义记忆读取） ----------

type getUserProfileTool struct {
	session *Session
}

func (t *getUserProfileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "get_user_profile",
		Description: "获取当前正在对话的小朋友的用户画像（名字、年龄段、感兴趣的话题）。" +
			"这是长期记忆：跨会话都有效。当需要称呼孩子的名字、选择适合难度的内容、或聊他感兴趣的话题时使用。",
		Parameters: `{"type":"object","properties":{}}`,
	}
}

func (t *getUserProfileTool) Execute(_ context.Context, _ string) (string, error) {
	b, err := json.Marshal(t.session.userProfile)
	if err != nil {
		return "获取用户画像失败。", nil
	}
	return string(b), nil
}

// ---------- 工具 3：update_user_profile（长期语义记忆写入） ----------

type updateUserProfileTool struct {
	session *Session
}

func (t *updateUserProfileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "update_user_profile",
		Description: "当孩子告诉你 TA 的名字、年龄，或喜欢的话题/动物/颜色时，调用此工具记住，" +
			"下次对话就能个性化（长期记忆）。参数均可选：name 名字、age_group 年龄段('6-8'或'9-12')、" +
			"preferred_topics 兴趣话题数组。只传你想更新的字段。",
		Parameters: `{"type":"object","properties":{"name":{"type":"string"},"age_group":{"type":"string","description":"6-8 或 9-12"},"preferred_topics":{"type":"array","items":{"type":"string"}}}}`,
	}
}

func (t *updateUserProfileTool) Execute(_ context.Context, args string) (string, error) {
	var p struct {
		Name           string   `json:"name"`
		AgeGroup       string   `json:"age_group"`
		PreferredTopics []string `json:"preferred_topics"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "参数解析失败，未更新画像。", nil
	}
	// 增量合并：只更新传了的字段
	if p.Name != "" {
		t.session.userProfile.Name = p.Name
	}
	if p.AgeGroup != "" {
		t.session.userProfile.AgeGroup = p.AgeGroup
	}
	if len(p.PreferredTopics) > 0 {
		t.session.userProfile.PreferredTopics = p.PreferredTopics
	}
	t.session.saveUserProfile() // 持久化到长期记忆存储（Redis/内存），忽略错误
	b, _ := json.Marshal(t.session.userProfile)
	return fmt.Sprintf("已记住小朋友的画像：%s", string(b)), nil
}

// ---------- 工具 4：get_learning_report（情景记忆读取） ----------

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

// ---------- 工具 5：start_pronunciation_practice（异步交互：开始跟读） ----------

type startPronunciationPracticeTool struct {
	svc     *PronunciationService
	session *Session
}

func (t *startPronunciationPracticeTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "start_pronunciation_practice",
		Description: "开始一次发音跟读练习。参数 unit_id（从 list_pronunciation_units 获取），item_index 可选（默认当前项）。" +
			"调用后系统会准备好示范内容并进入'等待跟读'状态：请对孩子说'来，跟着我说：<内容>'，等 TA 跟读后系统会自动评测并给你反馈。" +
			"通常紧接着调用 assess_pronunciation 明确进入等待。",
		Parameters: `{"type":"object","properties":{"unit_id":{"type":"string","description":"练习单元 ID"},"item_index":{"type":"integer","description":"练习项序号，可省略使用当前项"}}}`,
	}
}

func (t *startPronunciationPracticeTool) Execute(ctx context.Context, args string) (string, error) {
	if t.svc == nil {
		return "发音练习服务不可用。", nil
	}
	var p struct {
		UnitID    string `json:"unit_id"`
		ItemIndex int    `json:"item_index"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	if p.UnitID == "" {
		return "缺少 unit_id，请先调用 list_pronunciation_units 获取单元 ID。", nil
	}
	// 参数校验（P4 儿童安全/参数白名单）：unit_id 必须真实存在，避免 LLM 传入任意字符串。
	category := "read_word"
	valid := false
	for _, u := range t.svc.GetUnitList(ctx, "") {
		if u.ID == p.UnitID {
			valid = true
			if u.Type == "sentence" {
				category = "read_sentence"
			}
			break
		}
	}
	if !valid {
		return "unit_id 不存在，请先调用 list_pronunciation_units 获取有效的单元 ID。", nil
	}
	resp, err := t.svc.StartSession(ctx, &PronunciationStartSessionRequest{UserID: t.session.userID, UnitID: p.UnitID})
	if err != nil {
		return fmt.Sprintf("开始发音练习失败：%s", err.Error()), nil
	}
	if resp == nil || resp.CurrentItem == nil {
		return "该单元暂无可练习的内容。", nil
	}
	item := resp.CurrentItem
	// 进入"等待跟读"状态（控制权让渡）：下一句 ASR 音频将被截留并评测
	t.session.armFollowRead(item.Content, category, resp.SessionID, item.ItemIndex)

	out := map[string]interface{}{
		"content":           item.Content,
		"standard_audio_url": item.StandardAudioURL,
		"item_index":        item.ItemIndex,
		"total_items":       item.TotalItems,
		"is_last":           item.IsLast,
		"message":           "请让孩子跟读以下内容，我将等待音频并评测：" + item.Content,
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// ---------- 工具 6：assess_pronunciation（异步交互：等待并评测跟读） ----------

type assessPronunciationTool struct {
	session *Session
}

func (t *assessPronunciationTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "assess_pronunciation",
		Description: "当孩子刚跟读了一句后调用，让系统明确进入'等待下一句音频'状态并对跟读发音评分。" +
			"一般紧随 start_pronunciation_practice 使用；若尚未选择练习内容则先调用 start_pronunciation_practice。" +
			"调用后无需再说多余的话，静候音频即可。",
		Parameters: `{"type":"object","properties":{}}`,
	}
}

func (t *assessPronunciationTool) Execute(_ context.Context, _ string) (string, error) {
	target, cat, armed := t.session.followReadTarget()
	if !armed || target == "" {
		return "请先调用 start_pronunciation_practice 选择练习内容，我再等待孩子跟读并评测。", nil
	}
	// 重新确认等待状态（幂等），保证控制权已让渡
	t.session.armFollowRead(target, cat, t.session.practiceSessionID, t.session.practiceItemIndex)
	return "已准备好，请让孩子跟读，我会在下一句音频到达时自动评测并给出反馈。", nil
}

// ---------- 工具 7：set_session_goal（多轮目标规划） ----------

type setGoalTool struct {
	session *Session
}

func (t *setGoalTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "set_session_goal",
		Description: "为本次会话设定一个明确的教学目标（多轮目标规划）。" +
			"例如\"带孩子练 3 个动物单词并完成一次评测\"。设定后，龙宝 OK 会围绕这个目标主动引导孩子，" +
			"并在孩子走神/静默时主动推进。参数 description 目标描述；estimated_steps 可选，预计需要的步骤数。",
		Parameters: `{"type":"object","properties":{"description":{"type":"string","description":"本次会话目标的自然语言描述"},"estimated_steps":{"type":"integer","description":"预计需要的步骤数，可省略"}}}`,
	}
}

func (t *setGoalTool) Execute(_ context.Context, args string) (string, error) {
	var p struct {
		Description     string `json:"description"`
		EstimatedSteps  int    `json:"estimated_steps"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "参数解析失败，未设定目标。", nil
	}
	if p.Description == "" {
		return "目标描述不能为空，请提供 description。", nil
	}
	t.session.setGoal(p.Description, p.EstimatedSteps)
	return fmt.Sprintf("已设定本次会话目标：%s（状态：进行中）", p.Description), nil
}

// ---------- 工具 8：update_goal_progress（多轮目标规划） ----------

type updateGoalTool struct {
	session *Session
}

func (t *updateGoalTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "update_goal_progress",
		Description: "更新当前会话目标的进度。刚完成一步练习时，用 completed_step 记录这一步做了什么；" +
			"目标全部达成时用 status='done' 标记完成，或放弃时用 status='abandoned'。" +
			"这帮助龙宝 OK 在多轮对话中记住\"我们练到哪了\"，避免重复或遗漏。",
		Parameters: `{"type":"object","properties":{"completed_step":{"type":"string","description":"刚完成的步骤描述，如'练了单词 apple'"},"status":{"type":"string","description":"目标状态：done 或 abandoned，可省略"}}}`,
	}
}

func (t *updateGoalTool) Execute(_ context.Context, args string) (string, error) {
	var p struct {
		CompletedStep string `json:"completed_step"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "参数解析失败，未更新进度。", nil
	}
	t.session.updateGoalProgress(p.CompletedStep, p.Status)
	g := t.session.goalState
	if g == nil {
		return "已更新进度（当前无明确目标）。", nil
	}
	return fmt.Sprintf("目标进度已更新：%s | 已完成 %d 步 | 状态：%s",
		g.Description, len(g.DoneSteps), g.Status), nil
}

// registerAgentTools 按"依赖非空"原则，把工具注册进 Agent 注册表。
// 任何 service 为 nil 时跳过对应工具；注册表为空 → ReAct 循环退化为纯对话（与 P0 一致）。
func (s *Session) registerAgentTools(registry *agentpkg.Registry) {
	if s.pronunciationService != nil {
		registry.Register(&listUnitsTool{svc: s.pronunciationService})
		// 异步交互：开始跟读（依赖发音练习服务）
		registry.Register(&startPronunciationPracticeTool{svc: s.pronunciationService, session: s})
	}
	if s.evalProvider != nil {
		// 异步交互：等待并评测跟读（依赖语音评测服务）
		registry.Register(&assessPronunciationTool{session: s})
	}
	// 长期语义记忆：读取 / 写入（始终注册；profileStore 为 nil 时退化为内存桩）
	registry.Register(&getUserProfileTool{session: s})
	registry.Register(&updateUserProfileTool{session: s})
	if s.reportService != nil {
		registry.Register(&getLearningReportTool{svc: s.reportService, userID: s.userID})
	}
	// 多轮目标规划（P3）：目标管理工具始终注册，无外部依赖
	registry.Register(&setGoalTool{session: s})
	registry.Register(&updateGoalTool{session: s})
}
