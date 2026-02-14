// Package evaluation 提供评分与分级逻辑
package evaluation

// Scorer 评分器
type Scorer struct {
	// 评分权重配置
	weights ScoringWeights
}

// ScoringWeights 评分权重
type ScoringWeights struct {
	Accuracy     float64 `json:"accuracy"`     // 准确度权重
	Fluency      float64 `json:"fluency"`      // 流利度权重
	Completeness float64 `json:"completeness"` // 完整度权重
	Intonation   float64 `json:"intonation"`   // 语调权重
}

// NewScorer 创建评分器
func NewScorer() *Scorer {
	return &Scorer{
		weights: ScoringWeights{
			Accuracy:     0.4,
			Fluency:      0.3,
			Completeness: 0.2,
			Intonation:   0.1,
		},
	}
}

// CalculateOverallScore 计算综合评分
func (s *Scorer) CalculateOverallScore(result *RawEvaluationResult) float64 {
	// TODO: 实现综合评分计算
	// score = accuracy * w1 + fluency * w2 + completeness * w3 + intonation * w4
	return 0.0
}

// GetLevel 根据分数获取等级
func (s *Scorer) GetLevel(score float64) string {
	switch {
	case score >= 90:
		return "A" // 优秀
	case score >= 80:
		return "B" // 良好
	case score >= 70:
		return "C" // 中等
	case score >= 60:
		return "D" // 及格
	default:
		return "E" // 需要提高
	}
}

// RawEvaluationResult 原始评测结果（来自讯飞 API）
type RawEvaluationResult struct {
	Accuracy     float64         `json:"accuracy"`
	Fluency      float64         `json:"fluency"`
	Completeness float64         `json:"completeness"`
	Intonation   float64         `json:"intonation"`
	Words        []WordResult    `json:"words"`
	Phonemes     []PhonemeResult `json:"phonemes"`
}

// WordResult 单词级评测结果
type WordResult struct {
	Word      string  `json:"word"`
	Score     float64 `json:"score"`
	StartTime int     `json:"start_time"`
	EndTime   int     `json:"end_time"`
}

// PhonemeResult 音素级评测结果
type PhonemeResult struct {
	Phoneme   string  `json:"phoneme"`
	Score     float64 `json:"score"`
	StartTime int     `json:"start_time"`
	EndTime   int     `json:"end_time"`
}
