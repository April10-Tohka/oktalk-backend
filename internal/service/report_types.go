package service

// WeeklyReportData 周报完整 JSON（存入 learning_reports.content）
type WeeklyReportData struct {
	WeekStart string           `json:"week_start"`
	WeekEnd   string           `json:"week_end"`
	Activity  ReportActivity   `json:"activity"`
	Radar     ReportRadar      `json:"radar"`
	HardWords []ReportHardWord `json:"hard_words"`
	Scene     ReportScene      `json:"scene"`
	Encourage ReportEncourage  `json:"encourage"`
	FullReport ReportFullContent `json:"full_report"`
}

// ReportActivity 学习活跃度
type ReportActivity struct {
	EvaluationCount   int `json:"evaluation_count"`
	ConversationCount int `json:"conversation_count"`
	PersistenceDays   int `json:"persistence_days"`
}

// ReportRadar 能力雷达（百分制 0-100）
type ReportRadar struct {
	AccuracyScore  int `json:"accuracy_score"`
	FluencyScore   int `json:"fluency_score"`
	IntegrityScore int `json:"integrity_score"`
	StandardScore  int `json:"standard_score"`
}

// ReportHardWord 难词卡片
type ReportHardWord struct {
	Word     string `json:"word"`
	Count    int    `json:"count"`
	AudioURL string `json:"audio_url"`
}

// ReportScene 场景对话表现
type ReportScene struct {
	PassRate        int `json:"pass_rate"`
	CompletedScenes int `json:"completed_scenes"`
}

// ReportEncourage 鼓励卡片
type ReportEncourage struct {
	EncourageText string `json:"encourage_text"`
}

// ReportFullContent LLM 完整报告
type ReportFullContent struct {
	Summary      string   `json:"summary"`
	Strengths    []string `json:"strengths"`
	Improvements []string `json:"improvements"`
}

// WeeklyReportListItem 列表摘要
type WeeklyReportListItem struct {
	ReportID          string `json:"report_id"`
	WeekStart         string `json:"week_start"`
	WeekEnd           string `json:"week_end"`
	CreatedAt         string `json:"created_at"`
	AccuracyScore     int    `json:"accuracy_score"`
	PersistenceDays   int    `json:"persistence_days"`
	EvaluationCount   int    `json:"evaluation_count"`
}
