package xf

// webSocketFrame WebSocket 数据帧
type webSocketFrame struct {
	Common   commonParams   `json:"common"`
	Business businessParams `json:"business"`
	Data     dataParams     `json:"data"`
}

// commonParams 公共参数
type commonParams struct {
	AppID string `json:"app_id"`
}

// businessParams 业务参数
type businessParams struct {
	Category    string `json:"category"`     // 评测类型
	ResultLevel string `json:"result_level"` // 结果级别
	AufFormat   string `json:"auf_format"`   // 音频格式
	Tte         string `json:"tte"`          // 文本编码
	Text        string `json:"text"`         // 评测文本
}

// dataParams 数据参数
type dataParams struct {
	Status int    `json:"status"` // 0:第一帧 1:中间帧 2:最后一帧
	Data   string `json:"data"`   // Base64 编码的音频数据
}

// responseFrame 响应数据帧
type responseFrame struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Sid     string        `json:"sid"`
	Data    *responseData `json:"data"`
}

// responseData 响应数据
type responseData struct {
	Status int    `json:"status"`
	Data   string `json:"data"` // Base64 编码的评测结果
}

// assessmentCategory 评测类型
type assessmentCategory string

const (
	// ReadWord 单词朗读
	ReadWord assessmentCategory = "read_word"
	// ReadSentence 句子朗读
	ReadSentence assessmentCategory = "read_sentence"
	// ReadChapter 篇章朗读
	ReadChapter assessmentCategory = "read_chapter"
)
