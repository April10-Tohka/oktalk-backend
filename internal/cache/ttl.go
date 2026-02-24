package cache

import "time"

// ===================== TTL 常量 =====================

const (
	// TTLTaskMeta 任务元数据 24h
	TTLTaskMeta = 24 * time.Hour

	// TTLChatResult 对话结果 7天
	TTLChatResult = 7 * 24 * time.Hour

	// TTLEvalResult 评测结果 30天
	TTLEvalResult = 30 * 24 * time.Hour

	// TTLReportResult 报告结果 90天
	TTLReportResult = 90 * 24 * time.Hour

	// TTLHistoryList 历史列表 1h
	TTLHistoryList = 1 * time.Hour

	// TTLReportHistory 报告历史列表 6h
	TTLReportHistory = 6 * time.Hour

	// TTLDemoAudio 示范音频 365天
	TTLDemoAudio = 365 * 24 * time.Hour

	// TTLLLMFeedback LLM 反馈缓存 30天
	TTLLLMFeedback = 30 * 24 * time.Hour

	// TTLLLMReport LLM 报告缓存 7天
	TTLLLMReport = 7 * 24 * time.Hour

	// TTLTTSAudio TTS 音频缓存 30天
	TTLTTSAudio = 30 * 24 * time.Hour

	// TTLChatSession 会话记录 24h
	TTLChatSession = 24 * time.Hour

	// TTLUserInfo 用户信息 1h
	TTLUserInfo = 1 * time.Hour

	// TTLTextContent 学习文本 永久（传 0 时不调用 Expire）
	TTLTextContent = 0
)
