package domain

import "context"

// EvaluationProvider 语音评测服务提供者接口
// 封装科大讯飞等语音评测 API
type EvaluationProvider interface {
	// Assess 执行语音评测
	// text: 评测目标文本, audioData: 音频二进制数据
	Assess(ctx context.Context, text string, audioData []byte, category string) (*EvaluationResult, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// EvaluationResult 语音评测结果（领域层定义）
type EvaluationResult struct {
	Sid            string                 `json:"sid"`             // 讯飞语音评测会话 ID
	RawXML         string                 `json:"raw_xml"`         // 原始 XML 结果
	IsRejected     bool                   `json:"is_rejected"`     // 是否被拒绝
	ExceptInfo     string                 `json:"except_info"`     // 异常信息
	TotalScore     float64                `json:"total_score"`     // 综合评分
	AccuracyScore  float64                `json:"accuracy_score"`  // 准确度评分
	FluencyScore   float64                `json:"fluency_score"`   // 流利度评分
	IntegrityScore float64                `json:"integrity_score"` // 完整度评分
	StandardScore  float64                `json:"standard_score"`  // 标准度评分
	Words          []WordEvaluationResult `json:"words"`           // 单词级结果
}

// WordEvaluationResult 单词级评测结果
type WordEvaluationResult struct {
	Word      string                    `json:"word"`
	Score     float64                   `json:"score"`
	BeginTime int                       `json:"begin_time"`
	EndTime   int                       `json:"end_time"`
	DpMessage int                       `json:"dp_message"` // 增漏读信息
	Phonemes  []PhonemeEvaluationResult `json:"phonemes"`
}

// PhonemeEvaluationResult 音素级评测结果
type PhonemeEvaluationResult struct {
	Phoneme   string  `json:"phoneme"`
	Score     float64 `json:"score"`
	BeginTime int     `json:"begin_time"`
	EndTime   int     `json:"end_time"`
}
