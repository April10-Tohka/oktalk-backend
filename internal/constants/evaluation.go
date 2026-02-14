// Package constants 定义评测相关常量
package constants

// 评测等级
const (
	LevelA = "A" // 优秀 (90-100)
	LevelB = "B" // 良好 (80-89)
	LevelC = "C" // 中等 (70-79)
	LevelD = "D" // 及格 (60-69)
	LevelE = "E" // 需要提高 (0-59)
)

// 评测类型
const (
	EvaluationTypeWord      = "word"      // 单词
	EvaluationTypeSentence  = "sentence"  // 句子
	EvaluationTypeParagraph = "paragraph" // 段落
)

// 评测状态
const (
	EvaluationStatusPending    = 0 // 待处理
	EvaluationStatusSuccess    = 1 // 成功
	EvaluationStatusFailed     = 2 // 失败
	EvaluationStatusProcessing = 3 // 处理中
)

// 评分维度
const (
	ScoreDimensionAccuracy     = "accuracy"     // 准确度
	ScoreDimensionFluency      = "fluency"      // 流利度
	ScoreDimensionCompleteness = "completeness" // 完整度
	ScoreDimensionIntonation   = "intonation"   // 语调
)

// 评分权重
var ScoreWeights = map[string]float64{
	ScoreDimensionAccuracy:     0.4,
	ScoreDimensionFluency:      0.3,
	ScoreDimensionCompleteness: 0.2,
	ScoreDimensionIntonation:   0.1,
}

// 难度等级
const (
	DifficultyBeginner          = "beginner"           // 初级
	DifficultyElementary        = "elementary"         // 基础
	DifficultyIntermediate      = "intermediate"       // 中级
	DifficultyUpperIntermediate = "upper_intermediate" // 中高级
	DifficultyAdvanced          = "advanced"           // 高级
)

// 场景分类
const (
	ScenarioDaily     = "daily"     // 日常对话
	ScenarioBusiness  = "business"  // 商务英语
	ScenarioTravel    = "travel"    // 旅行英语
	ScenarioAcademic  = "academic"  // 学术英语
	ScenarioInterview = "interview" // 面试英语
)

// 音频限制
const (
	MaxAudioDurationSeconds = 300              // 最大音频时长（秒）
	MaxAudioSizeBytes       = 10 * 1024 * 1024 // 最大音频大小（10MB）
	MinAudioDurationSeconds = 1                // 最小音频时长（秒）
)

// 文本限制
const (
	MaxTextLength = 5000 // 最大文本长度
	MinTextLength = 1    // 最小文本长度
)

// GetLevelByScore 根据分数获取等级
func GetLevelByScore(score float64) string {
	switch {
	case score >= 90:
		return LevelA
	case score >= 80:
		return LevelB
	case score >= 70:
		return LevelC
	case score >= 60:
		return LevelD
	default:
		return LevelE
	}
}

// GetLevelDescription 获取等级描述
func GetLevelDescription(level string) string {
	descriptions := map[string]string{
		LevelA: "优秀 - 发音非常标准",
		LevelB: "良好 - 发音比较标准",
		LevelC: "中等 - 发音基本正确",
		LevelD: "及格 - 发音有待改进",
		LevelE: "需要提高 - 需要加强练习",
	}
	return descriptions[level]
}
