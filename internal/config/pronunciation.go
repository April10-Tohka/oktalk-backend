package config

// PronunciationUnitConfig 发音练习单元配置
type PronunciationUnitConfig struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"` // "word" / "sentence"
	Topic      string               `json:"topic"`
	Title      string               `json:"title"`
	CoverEmoji string               `json:"cover_emoji"`
	Items      []PronunciationItem  `json:"items"`
}

// PronunciationItem 单元内单条练习内容
type PronunciationItem struct {
	ID               int    `json:"id"`
	Content          string `json:"content"`
	StandardAudioURL string `json:"standard_audio_url"`
}
