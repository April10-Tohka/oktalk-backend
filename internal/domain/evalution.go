package domain

import "context"

// EvaluationProvider 语音评测服务提供者接口
// 封装科大讯飞等语音评测 API
type EvaluationProvider interface {
	// Assess 执行语音评测
	// text: 评测目标文本, audioData: 音频二进制数据
	Assess(ctx context.Context, text string, audioData []byte) (*EvaluationResult, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// EvaluationResult 语音评测结果（领域层定义）
type EvaluationResult struct {
	TotalScore   float64                `json:"total_score"`  // 综合评分
	Accuracy     float64                `json:"accuracy"`     // 准确度
	Fluency      float64                `json:"fluency"`      // 流利度
	Completeness float64                `json:"completeness"` // 完整度
	Intonation   float64                `json:"intonation"`   // 语调
	Words        []WordEvaluationResult `json:"words"`        // 单词级结果
}

// WordEvaluationResult 单词级评测结果
type WordEvaluationResult struct {
	Word      string                    `json:"word"`
	Score     float64                   `json:"score"`
	BeginTime int                       `json:"begin_time"`
	EndTime   int                       `json:"end_time"`
	Phonemes  []PhonemeEvaluationResult `json:"phonemes"`
}

// PhonemeEvaluationResult 音素级评测结果
type PhonemeEvaluationResult struct {
	Phoneme   string  `json:"phoneme"`
	Score     float64 `json:"score"`
	BeginTime int     `json:"begin_time"`
	EndTime   int     `json:"end_time"`
}
