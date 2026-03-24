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
	AccuracyScore  string `xml:"accuracy_score,attr"`  //准确度评分
	StandardScore  string `xml:"standard_score,attr"`  //标准度分
	FluencyScore   string `xml:"fluency_score,attr"`   //流畅度评分
	IntegrityScore string `xml:"integrity_score,attr"` //完整度评分
	TotalScore     string `xml:"total_score,attr"`     //篇章总分
	BegPos         string `xml:"beg_pos,attr"`         //篇章开始时间
	EndPos         string `xml:"end_pos,attr"`         //篇章结束时间
	Content        string `xml:"content,attr"`         //篇章内容
	ExceptInfo     string `xml:"except_info,attr"`     //异常信息
	IsRejected     string `xml:"is_rejected,attr"`     //是否被拒
	WordCount      string `xml:"word_count,attr"`      //篇章全部单词数量

	Sentences []xmlSentence `xml:"sentence"`
}

// xmlSentence 句子级结果
type xmlSentence struct {
	AccuracyScore string `xml:"accuracy_score,attr"`
	StandardScore string `xml:"standard_score,attr"`
	FluencyScore  string `xml:"fluency_score,attr"`
	TotalScore    string `xml:"total_score,attr"`
	BegPos        string `xml:"beg_pos,attr"`
	EndPos        string `xml:"end_pos,attr"`
	Content       string `xml:"content,attr"`
	WordCount     string `xml:"word_count,attr"` //句子全部单词数量

	Words []xmlWord `xml:"word"`
}

// xmlWord 单词级结果
type xmlWord struct {
	Content    string    `xml:"content,attr"`
	BegPos     string    `xml:"beg_pos,attr"`
	EndPos     string    `xml:"end_pos,attr"`
	DpMessage  string    `xml:"dp_message,attr"`  //单词增漏读信息
	TotalScore string    `xml:"total_score,attr"` //单词总分
	WerrMsg    string    `xml:"werr_msg,attr"`    //针对错误单词给出结果（正确不输出）
	Sylls      []xmlSyll `xml:"syll"`
}

// xmlSyll 音节级结果
type xmlSyll struct {
	Content    string `xml:"content,attr"`
	BegPos     string `xml:"beg_pos,attr"`
	EndPos     string `xml:"end_pos,attr"`
	SerrMsg    string `xml:"serr_msg,attr"`
	SyllAccent string `xml:"syll_accent,attr"`

	Phones []xmlPhone `xml:"phone"`
}

// xmlPhone 音素级结果
type xmlPhone struct {
	Content   string `xml:"content,attr"`
	BegPos    string `xml:"beg_pos,attr"`
	EndPos    string `xml:"end_pos,attr"`
	DpMessage string `xml:"dp_message,attr"`
}

// ===================== 内部 SDK 数据结构 =====================

// speechAssessRequest 语音评测请求
type speechAssessRequest struct {
	Text      string // 待评测文本
	AudioData []byte // 音频二进制数据
	Category  string // 题型: read_word / read_sentence / read_chapter
	Language  string // en_vip / cn_vip
}

// speechAssessResult 语音评测结果 (从 XML 解析后的结构)
type speechAssessResult struct {
	RawXML         string           // 原始 XML 结果 (可选，用于调试)
	TotalScore     float64          // 总分
	AccuracyScore  float64          // 准确度评分
	FluencyScore   float64          // 流畅度评分
	IntegrityScore float64          // 完整度评分
	StandardScore  float64          // 标准度评分
	IsRejected     bool             // 是否被拒绝
	ExceptInfo     int              // 异常信息代码 (0=正常, 28673=无语音/音量小, 28676=乱说, 28689=无音频输入)
	BegPos         int              // 音频起始位置 (ms)
	EndPos         int              // 音频结束位置 (ms)
	Content        string           // 评测文本内容
	WordCount      int              // 单词数量
	Sentences      []sentenceResult // 句子详情
}

// sentenceResult 句子级评测结果
type sentenceResult struct {
	AccuracyScore float64      // 准确度评分
	StandardScore float64      // 标准度评分
	FluencyScore  float64      // 流畅度评分
	TotalScore    float64      // 总分
	BegPos        int          // 起始位置 (ms)
	EndPos        int          // 结束位置 (ms)
	Content       string       // 句子内容
	WordCount     int          // 单词数量
	Words         []wordResult // 单词详情
}

// wordResult 单词级评测结果
type wordResult struct {
	Content    string       // 单词内容 (如 "hello" 或 "sil" 表示静音)
	BegPos     int          // 起始位置 (ms)
	EndPos     int          // 结束位置 (ms)
	DpMessage  int          // 发音诊断信息 (0=正常)
	TotalScore float64      // 单词得分 (sil 为 0)
	WerrMsg    string       // 错误信息
	Sylls      []syllResult // 音节详情
}

// syllResult 音节级评测结果
type syllResult struct {
	Content    string        // 音节内容 (如 "hh eh", "l ow")
	BegPos     int           // 起始位置 (ms)
	EndPos     int           // 结束位置 (ms)
	SerrMsg    int           // 音节错误码 (0=正常)
	SyllAccent int           // 重音标记 (0=无重音, 1=有重音)
	Phones     []phoneResult // 音素详情
}

// phoneResult 音素级评测结果
type phoneResult struct {
	Content   string // 音素内容 (如 "hh", "eh", "l", "ow")
	BegPos    int    // 起始位置 (ms)
	EndPos    int    // 结束位置 (ms)
	DpMessage int    // 发音诊断 (0=正常)
}

// assessmentCategory 评测类型常量
type assessmentCategory string

const (
	ReadWord     assessmentCategory = "read_word"
	ReadSentence assessmentCategory = "read_sentence"
	ReadChapter  assessmentCategory = "read_chapter"
)
