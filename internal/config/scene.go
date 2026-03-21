package config

// SceneConfig 单个场景的完整配置
type SceneConfig struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	CoverEmoji   string      `json:"cover_emoji"`
	Steps        []SceneStep `json:"steps"`
	SummaryIntro string      `json:"summary_intro"`
	SummaryItems []string    `json:"summary_items"`
}

// SceneStep 场景中的单个步骤
type SceneStep struct {
	ID                int      `json:"id"`
	Question          string   `json:"question"`
	QuestionAudioText string   `json:"question_audio_text"`
	Expected          []string `json:"expected"`
	Hints             []string `json:"hints"`
	Teach             string   `json:"teach"`
}
