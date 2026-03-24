package domain

import "context"

// EvaluationProvider 语音评测服务提供者接口
type EvaluationProvider interface {
	// Assess 执行语音评测
	// text: 目标文本（单词或整句）；
	// audioData: 音频 PCM 等数据；
	// category: "read_word" 或 "read_sentence"
	Assess(ctx context.Context, text string, audioData []byte, category string) (*EvaluationResult, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// EvaluationResult 一次发音评测的聚合结果。
// 单词题：通常 Words 仅一项（或按厂商拆分的若干可见单元）；
// 句子题：Words 为该句内各词（含 sil 等可再在 Adapter 中过滤），供问题词汇总与星级展示。
type EvaluationResult struct {
	RawXML         string                 `json:"raw_xml"`         // 原始结果（如 XML），便于排错与审计
	IsRejected     bool                   `json:"is_rejected"`     // 是否被拒绝
	ExceptInfo     string                 `json:"except_info"`     // 异常/拒识说明（厂商编码由 Adapter 透传为可读信息亦可）
	TotalScore     float64                `json:"total_score"`     // 综合评分（如 0～5）
	AccuracyScore  float64                `json:"accuracy_score"`  // 准确度
	FluencyScore   float64                `json:"fluency_score"`   // 流利度
	IntegrityScore float64                `json:"integrity_score"` // 完整度（句子题更有意义）
	StandardScore  float64                `json:"standard_score"`  // 标准度
	Words          []WordEvaluationResult `json:"words"`           // 单词级结果（句子题为多词列表）
}

// WordEvaluationResult 单词级结果（儿童端可高亮「哪个词」需加强）。
type WordEvaluationResult struct {
	Word      string                    `json:"word"`
	Score     float64                   `json:"score"`
	BeginTime int                       `json:"begin_time"`
	EndTime   int                       `json:"end_time"`
	DpMessage int                       `json:"dp_message"` // 增漏读等（0 表示正常）
	Phonemes  []PhonemeEvaluationResult `json:"phonemes"`   // 音素级（可选，用于更细反馈）
}

// PhonemeEvaluationResult 音素级结果。
type PhonemeEvaluationResult struct {
	Phoneme   string  `json:"phoneme"`
	Score     float64 `json:"score"`
	BeginTime int     `json:"begin_time"`
	EndTime   int     `json:"end_time"`
}
