// Package constants 定义反馈相关常量
package constants

// 反馈等级
const (
	FeedbackLevelExcellent        = "excellent"         // 优秀
	FeedbackLevelGood             = "good"              // 良好
	FeedbackLevelAverage          = "average"           // 中等
	FeedbackLevelBelowAverage     = "below_average"     // 待提高
	FeedbackLevelNeedsImprovement = "needs_improvement" // 需要改进
)

// 反馈类型
const (
	FeedbackTypeGeneral       = "general"       // 综合反馈
	FeedbackTypePronunciation = "pronunciation" // 发音反馈
	FeedbackTypeIntonation    = "intonation"    // 语调反馈
	FeedbackTypeRhythm        = "rhythm"        // 节奏反馈
)

// 问题严重程度
const (
	SeverityHigh   = "high"   // 严重
	SeverityMedium = "medium" // 中等
	SeverityLow    = "low"    // 轻微
)

// 问题类型
const (
	ProblemTypeSubstitution = "substitution" // 替换错误
	ProblemTypeOmission     = "omission"     // 省略错误
	ProblemTypeInsertion    = "insertion"    // 插入错误
	ProblemTypeStress       = "stress"       // 重音错误
	ProblemTypeIntonation   = "intonation"   // 语调错误
)

// 反馈语气
const (
	ToneEncouraging = "encouraging" // 鼓励性
	TonePositive    = "positive"    // 积极性
	ToneSupportive  = "supportive"  // 支持性
	ToneHelpful     = "helpful"     // 帮助性
	TonePatient     = "patient"     // 耐心性
)

// FeedbackLevelConfig 反馈等级配置
type FeedbackLevelConfig struct {
	Level       string
	MinScore    float64
	MaxScore    float64
	Description string
	Tone        string
	Focus       string
}

// FeedbackLevels 反馈等级配置列表
var FeedbackLevels = []FeedbackLevelConfig{
	{
		Level:       FeedbackLevelExcellent,
		MinScore:    90,
		MaxScore:    100,
		Description: "发音非常标准",
		Tone:        ToneEncouraging,
		Focus:       "维持优势，挑战更高难度",
	},
	{
		Level:       FeedbackLevelGood,
		MinScore:    80,
		MaxScore:    89,
		Description: "发音良好",
		Tone:        TonePositive,
		Focus:       "细节改进",
	},
	{
		Level:       FeedbackLevelAverage,
		MinScore:    70,
		MaxScore:    79,
		Description: "发音基本正确",
		Tone:        ToneSupportive,
		Focus:       "常见错误纠正",
	},
	{
		Level:       FeedbackLevelBelowAverage,
		MinScore:    60,
		MaxScore:    69,
		Description: "发音需要改进",
		Tone:        ToneHelpful,
		Focus:       "基础音素练习",
	},
	{
		Level:       FeedbackLevelNeedsImprovement,
		MinScore:    0,
		MaxScore:    59,
		Description: "发音需要大量练习",
		Tone:        TonePatient,
		Focus:       "从基础开始练习",
	},
}

// GetFeedbackLevel 根据分数获取反馈等级
func GetFeedbackLevel(score float64) *FeedbackLevelConfig {
	for _, config := range FeedbackLevels {
		if score >= config.MinScore && score <= config.MaxScore {
			return &config
		}
	}
	return &FeedbackLevels[len(FeedbackLevels)-1]
}
