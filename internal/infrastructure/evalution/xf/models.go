package xf

import "encoding/xml"

// ===================== WebSocket 请求帧 =====================

// webSocketFrame WebSocket 数据帧（发送给讯飞 ISE API）
type webSocketFrame struct {
	Common   *commonParams   `json:"common,omitempty"`
	Business *businessParams `json:"business,omitempty"`
	Data     *dataParams     `json:"data"`
}

// commonParams 公共参数
type commonParams struct {
	AppID string `json:"app_id"`
}

// businessParams 业务参数
// 根据 cmd 不同阶段，部分字段可选
type businessParams struct {
	Sub          string `json:"sub,omitempty"`           // 服务类型: ise
	Ent          string `json:"ent,omitempty"`           // cn_vip / en_vip
	Category     string `json:"category,omitempty"`      // read_word / read_sentence / read_chapter
	Cmd          string `json:"cmd"`                     // ssb / auw
	Text         string `json:"text,omitempty"`          // 待评测文本 (UTF-8 BOM)
	Tte          string `json:"tte,omitempty"`           // 文本编码: utf-8
	TtpSkip      bool   `json:"ttp_skip,omitempty"`      // 跳过 ttp 阶段
	Aue          string `json:"aue,omitempty"`           // 音频格式: raw
	Auf          string `json:"auf,omitempty"`           // 采样率: audio/L16;rate=16000
	Rstcd        string `json:"rstcd,omitempty"`         // 返回格式: utf8
	Aus          int    `json:"aus,omitempty"`           // 音频状态: 1首帧 2中间 4尾帧
	Rst          string `json:"rst,omitempty"`           // 返回结果控制: entirety
	IseUnite     string `json:"ise_unite,omitempty"`     // 返回结果控制: 1
	Plev         string `json:"plev,omitempty"`          // 结果详情级别: 0
	ExtraAbility string `json:"extra_ability,omitempty"` // 拓展能力: multi_dimension
	Group        string `json:"group,omitempty"`         // 评测群体: pupil / youth / adult
}

// dataParams 数据参数
type dataParams struct {
	Status int    `json:"status"`         // 0:第一帧 1:中间帧 2:最后一帧
	Data   string `json:"data,omitempty"` // Base64 编码的音频数据
}

// ===================== WebSocket 响应帧 =====================

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
	Data   string `json:"data"` // Base64 编码的评测结果 (XML)
}

// ===================== XML 评测结果结构 (英文) =====================

// xmlResult 评测 XML 顶层结构
type xmlResult struct {
	XMLName      xml.Name      `xml:"xml_result"`
	ReadSentence *xmlReadBlock `xml:"read_sentence"`
	ReadWord     *xmlReadBlock `xml:"read_word"`
	ReadChapter  *xmlReadBlock `xml:"read_chapter"`
}

// xmlReadBlock 阅读题型块 (read_sentence / read_word / read_chapter)
type xmlReadBlock struct {
	Lan      string       `xml:"lan,attr"`
	Type     string       `xml:"type,attr"`
	Version  string       `xml:"version,attr"`
	RecPaper *xmlRecPaper `xml:"rec_paper"`
}

// xmlRecPaper 评测试卷
type xmlRecPaper struct {
	// 英文句子/篇章题型
	ReadSentence *xmlReadItem `xml:"read_sentence"`
	ReadWord     *xmlReadItem `xml:"read_word"`
	ReadChapter  *xmlReadItem `xml:"read_chapter"`
}

// xmlReadItem 评测项 (句子/单词/篇章级别)
type xmlReadItem struct {
	AccuracyScore  string `xml:"accuracy_score,attr"`
	StandardScore  string `xml:"standard_score,attr"`
	FluencyScore   string `xml:"fluency_score,attr"`
	IntegrityScore string `xml:"integrity_score,attr"`
	PhoneScore     string `xml:"phone_score,attr"`
	ToneScore      string `xml:"tone_score,attr"`
	TotalScore     string `xml:"total_score,attr"`
	BegPos         string `xml:"beg_pos,attr"`
	EndPos         string `xml:"end_pos,attr"`
	Content        string `xml:"content,attr"`
	ExceptInfo     string `xml:"except_info,attr"`
	IsRejected     string `xml:"is_rejected,attr"`
	TimeLen        string `xml:"time_len,attr"`

	Sentences []xmlSentence `xml:"sentence"`
}

// xmlSentence 句子级结果
type xmlSentence struct {
	AccuracyScore string `xml:"accuracy_score,attr"`
	StandardScore string `xml:"standard_score,attr"`
	FluencyScore  string `xml:"fluency_score,attr"`
	PhoneScore    string `xml:"phone_score,attr"`
	ToneScore     string `xml:"tone_score,attr"`
	TotalScore    string `xml:"total_score,attr"`
	BegPos        string `xml:"beg_pos,attr"`
	EndPos        string `xml:"end_pos,attr"`
	Content       string `xml:"content,attr"`
	TimeLen       string `xml:"time_len,attr"`

	Words []xmlWord `xml:"word"`
}

// xmlWord 单词级结果
type xmlWord struct {
	Content    string `xml:"content,attr"`
	Symbol     string `xml:"symbol,attr"`
	BegPos     string `xml:"beg_pos,attr"`
	EndPos     string `xml:"end_pos,attr"`
	TimeLen    string `xml:"time_len,attr"`
	DpMessage  string `xml:"dp_message,attr"`
	TotalScore string `xml:"total_score,attr"`

	Sylls []xmlSyll `xml:"syll"`
}

// xmlSyll 音节级结果
type xmlSyll struct {
	Content     string `xml:"content,attr"`
	Symbol      string `xml:"symbol,attr"`
	BegPos      string `xml:"beg_pos,attr"`
	EndPos      string `xml:"end_pos,attr"`
	TimeLen     string `xml:"time_len,attr"`
	DpMessage   string `xml:"dp_message,attr"`
	RecNodeType string `xml:"rec_node_type,attr"`
	SyllScore   string `xml:"syll_score,attr"`
	SerrMsg     string `xml:"serr_msg,attr"`
	SyllAccent  string `xml:"syll_accent,attr"`

	Phones []xmlPhone `xml:"phone"`
}

// xmlPhone 音素级结果
type xmlPhone struct {
	Content      string `xml:"content,attr"`
	BegPos       string `xml:"beg_pos,attr"`
	EndPos       string `xml:"end_pos,attr"`
	TimeLen      string `xml:"time_len,attr"`
	DpMessage    string `xml:"dp_message,attr"`
	RecNodeType  string `xml:"rec_node_type,attr"`
	IsYun        string `xml:"is_yun,attr"`
	PerrMsg      string `xml:"perr_msg,attr"`
	PerrLevelMsg string `xml:"perr_level_msg,attr"`
	MonoTone     string `xml:"mono_tone,attr"`
}

// ===================== 内部 SDK 数据结构 =====================

// speechAssessRequest 语音评测请求
type speechAssessRequest struct {
	Text      string // 待评测文本
	AudioData []byte // 音频二进制数据
	Category  string // 题型: read_word / read_sentence / read_chapter
	Language  string // en_vip / cn_vip
}

// speechAssessResult 语音评测结果 (从 XML 解析后的中间结构)
type speechAssessResult struct {
	TotalScore   float64
	Accuracy     float64
	Fluency      float64
	Completeness float64 // IntegrityScore
	Intonation   float64 // StandardScore (用作语调)
	Words        []wordResult
}

// wordResult 单词评测结果
type wordResult struct {
	Word      string
	Score     float64
	BeginTime int
	EndTime   int
	DpMessage int // 0正常 16漏读 32增读 64回读 128替换
	Phonemes  []phonemeResult
}

// phonemeResult 音素评测结果
type phonemeResult struct {
	Phoneme   string
	Score     float64
	BeginTime int
	EndTime   int
}

// assessmentCategory 评测类型常量
type assessmentCategory string

const (
	ReadWord     assessmentCategory = "read_word"
	ReadSentence assessmentCategory = "read_sentence"
	ReadChapter  assessmentCategory = "read_chapter"
)
