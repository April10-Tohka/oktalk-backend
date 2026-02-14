// Package evaluation 提供音素级分析逻辑
package evaluation

// Analyzer 音素分析器
type Analyzer struct{}

// NewAnalyzer 创建音素分析器
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzePhonemes 分析音素级发音问题
func (a *Analyzer) AnalyzePhonemes(result *RawEvaluationResult) *PhonemeAnalysis {
	// TODO: 实现音素分析
	// 1. 识别发音错误的音素
	// 2. 分类错误类型（替换、省略、插入）
	// 3. 生成改进建议
	return &PhonemeAnalysis{}
}

// AnalyzeProblems 分析发音问题
func (a *Analyzer) AnalyzeProblems(result *RawEvaluationResult) []*PronunciationProblem {
	// TODO: 实现问题分析
	return nil
}

// PhonemeAnalysis 音素分析结果
type PhonemeAnalysis struct {
	TotalPhonemes    int                `json:"total_phonemes"`
	CorrectPhonemes  int                `json:"correct_phonemes"`
	ErrorPhonemes    []PhonemeError     `json:"error_phonemes"`
	CommonErrors     []CommonError      `json:"common_errors"`
	Suggestions      []string           `json:"suggestions"`
}

// PhonemeError 音素错误
type PhonemeError struct {
	Phoneme      string `json:"phoneme"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	ErrorType    string `json:"error_type"` // substitution, omission, insertion
	Position     int    `json:"position"`
	Word         string `json:"word"`
}

// CommonError 常见错误模式
type CommonError struct {
	Pattern     string `json:"pattern"`
	Frequency   int    `json:"frequency"`
	Description string `json:"description"`
}

// PronunciationProblem 发音问题
type PronunciationProblem struct {
	Type        string   `json:"type"`        // phoneme, stress, intonation, rhythm
	Severity    string   `json:"severity"`    // high, medium, low
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Suggestions []string `json:"suggestions"`
}
