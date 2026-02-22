package llm

import "fmt"

// ReportStats 报告统计数据（传给 LLM Prompt）
type ReportStats struct {
	StartDate         string
	EndDate           string
	ConversationCount int
	EvaluationCount   int
	TotalMinutes      int
	ActiveDays        int
	AvgAccuracy       float64
	AvgFluency        float64
	AvgIntegrity      float64
	AvgOverall        float64
	SCount            int
	ACount            int
	BCount            int
	CCount            int
	PreviousScore     float64
	CurrentScore      float64
	ScoreChange       float64
	ProblemWords      map[string]int // word -> frequency
}

// LLMReportData LLM 返回的报告 JSON 结构
type LLMReportData struct {
	PeriodSummary     string   `json:"period_summary"`
	AbilityAnalysis   string   `json:"ability_analysis"`
	ProgressHighlight string   `json:"progress_highlight"`
	ImprovementAreas  []string `json:"improvement_areas"`
	Recommendations   []string `json:"recommendations"`
	EncouragementText string   `json:"encouragement_text"`
	Highlights        []string `json:"highlights"`
	SmallGoal         string   `json:"small_goal"`
	FullText          string   `json:"full_text"`
}

// GenerateReportPrompt 生成学习报告的 LLM Prompt（system + user）
func GenerateReportPrompt(stats *ReportStats) (system string, user string) {
	system = `你是一位专业的儿童英语学习顾问。请根据学习数据生成报告内容。
请只返回纯 JSON，不要有 markdown 代码块标记或其他内容。

JSON 格式要求：
{
  "period_summary": "本周期小结（50字，鼓励式语言，适合孩子阅读）",
  "ability_analysis": "能力表现分析（40字，说明准确度/流利度/完整度表现）",
  "progress_highlight": "进步亮点（30字，强调提升点）",
  "improvement_areas": ["改进点1（温和）", "改进点2（可选）"],
  "recommendations": ["具体建议1（可执行）", "具体建议2（可选）"],
  "encouragement_text": "一句话鼓励（15字内，充满正能量）",
  "highlights": ["亮点1（4字）", "亮点2（4字）"],
  "small_goal": "小目标（10字内，简单明确）",
  "full_text": "完整报告正文（100-150字，口语化，鼓励语气）"
}

关键要求：
1. 语气温暖、正向、鼓励
2. 不使用"差""不好"等负面词汇
3. 改进点要温和委婉
4. 建议要具体可执行（如"每天跟读 5 分钟"）
5. 适合 6-12 岁儿童理解`

	user = fmt.Sprintf(`【本周学习数据】
时间周期：%s 到 %s
对话次数：%d 次
评测次数：%d 次
学习时长：%d 分钟
活跃天数：%d 天

【能力评分（0-100）】
准确度：%.1f 分
流利度：%.1f 分
完整度：%.1f 分
综合分：%.1f 分

【评级分布】
S 级（优秀）：%d 次
A 级（良好）：%d 次
B 级（一般）：%d 次
C 级（需加强）：%d 次

【进步对比】
上周综合分：%.1f 分
本周综合分：%.1f 分
进步：%+.1f 分

【高频问题单词】
%s`,
		stats.StartDate, stats.EndDate,
		stats.ConversationCount, stats.EvaluationCount,
		stats.TotalMinutes, stats.ActiveDays,
		stats.AvgAccuracy, stats.AvgFluency, stats.AvgIntegrity, stats.AvgOverall,
		stats.SCount, stats.ACount, stats.BCount, stats.CCount,
		stats.PreviousScore, stats.CurrentScore, stats.ScoreChange,
		FormatProblemWords(stats.ProblemWords))

	return
}

// FormatProblemWords 格式化问题单词列表
func FormatProblemWords(words map[string]int) string {
	if len(words) == 0 {
		return "无"
	}
	var result string
	for word, freq := range words {
		result += fmt.Sprintf("- %s (出现 %d 次)\n", word, freq)
	}
	return result
}

// GetDefaultReportData 返回默认的报告数据（LLM 降级时使用）
func GetDefaultReportData() LLMReportData {
	return LLMReportData{
		PeriodSummary:     "这周你很努力地学习英语，继续加油！",
		AbilityAnalysis:   "各方面能力都在稳步提升中。",
		ProgressHighlight: "坚持练习就是最大的进步！",
		ImprovementAreas:  []string{"多练习发音"},
		Recommendations:   []string{"每天跟读 5 分钟"},
		EncouragementText: "你做得很棒，继续加油！",
		Highlights:        []string{"坚持学习", "勇敢开口"},
		SmallGoal:         "每天跟读 5 分钟",
		FullText:          "这周你认真完成了学习任务，英语能力正在稳步提升。继续保持每天练习的好习惯，你一定会越来越棒的！",
	}
}
