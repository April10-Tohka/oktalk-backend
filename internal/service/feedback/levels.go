// Package feedback 提供反馈分级规则
package feedback

// Levels 反馈分级规则
type Levels struct {
	rules []LevelRule
}

// LevelRule 分级规则
type LevelRule struct {
	Level       string  `json:"level"`
	MinScore    float64 `json:"min_score"`
	MaxScore    float64 `json:"max_score"`
	Description string  `json:"description"`
	Tone        string  `json:"tone"`        // 反馈语气
	Focus       string  `json:"focus"`       // 反馈重点
}

// NewLevels 创建分级规则
func NewLevels() *Levels {
	return &Levels{
		rules: []LevelRule{
			{
				Level:       "excellent",
				MinScore:    90,
				MaxScore:    100,
				Description: "发音非常标准",
				Tone:        "encouraging",
				Focus:       "维持优势，挑战更高难度",
			},
			{
				Level:       "good",
				MinScore:    80,
				MaxScore:    89,
				Description: "发音良好",
				Tone:        "positive",
				Focus:       "细节改进",
			},
			{
				Level:       "average",
				MinScore:    70,
				MaxScore:    79,
				Description: "发音基本正确",
				Tone:        "supportive",
				Focus:       "常见错误纠正",
			},
			{
				Level:       "below_average",
				MinScore:    60,
				MaxScore:    69,
				Description: "发音需要改进",
				Tone:        "helpful",
				Focus:       "基础音素练习",
			},
			{
				Level:       "needs_improvement",
				MinScore:    0,
				MaxScore:    59,
				Description: "发音需要大量练习",
				Tone:        "patient",
				Focus:       "从基础开始练习",
			},
		},
	}
}

// GetLevel 根据分数获取等级
func (l *Levels) GetLevel(score float64) *LevelRule {
	for _, rule := range l.rules {
		if score >= rule.MinScore && score <= rule.MaxScore {
			return &rule
		}
	}
	return &l.rules[len(l.rules)-1]
}

// GetFeedbackTemplate 获取反馈模板
func (l *Levels) GetFeedbackTemplate(level string) string {
	templates := map[string]string{
		"excellent":         "太棒了！您的发音非常标准。{details}继续保持！",
		"good":              "做得很好！{details}再接再厉！",
		"average":           "不错的尝试！{details}多加练习会更好。",
		"below_average":     "继续加油！{details}我们一起努力改进。",
		"needs_improvement": "别灰心！{details}每天练习一点点，进步看得见。",
	}
	return templates[level]
}
