package xf

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// XFEvaluationAdapter 科大讯飞语音评测适配器
// 实现 domain.EvaluationProvider 接口
type XFEvaluationAdapter struct {
	appID     string
	apiKey    string
	apiSecret string
}

// 编译时检查：确保 XFEvaluationAdapter 实现了 domain.EvaluationProvider 接口
var _ domain.EvaluationProvider = (*XFEvaluationAdapter)(nil)

// NewXFEvaluationAdapter 创建科大讯飞语音评测适配器
func NewXFEvaluationAdapter(cfg config.XunFeiConfig) *XFEvaluationAdapter {
	return &XFEvaluationAdapter{
		appID:     cfg.AppID,
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
	}
}

// Assess 执行语音评测
func (a *XFEvaluationAdapter) Assess(ctx context.Context, text string, audioData []byte, category string) (*domain.EvaluationResult, error) {
	logger.InfoContext(ctx, "xf evaluation: starting assess",
		"text", text,
		"audio_bytes", len(audioData),
		"category", category)

	req := &speechAssessRequest{
		Text:      text,
		AudioData: audioData,
		Category:  category,
		Language:  "en_vip",
	}
	// 1. 构建鉴权 URL 并建立 WebSocket 连接
	wsURL, err := a.buildAuthURL()
	if err != nil {
		return nil, fmt.Errorf("build auth url: %w", err)
	}
	logger.InfoContext(ctx, "xf ise building auth url successfully")

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()
	logger.InfoContext(ctx, "xf ise connected successfully")

	// 2. 发送 SSB 参数帧（cmd=ssb, data.status=0）
	if err := a.sendSSBFrame(ctx, conn, req); err != nil {
		return nil, fmt.Errorf("send ssb frame: %w", err)
	}

	// 3. 分片发送音频（cmd=auw）
	if err := a.sendAudioFrames(ctx, conn, req.AudioData); err != nil {
		return nil, fmt.Errorf("send audio frames: %w", err)
	}

	logger.InfoContext(ctx, "xf ise all frames sent successfully, waiting for result")

	// 4. 接收评测结果
	resultXML, err := a.receiveResult(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("receive result: %w", err)
	}
	logger.InfoContext(ctx, "xf ise receive result successfully")
	// 5. 解析 XML 评测结果
	result, err := a.parseXMLResult(resultXML)
	if err != nil {
		return nil, fmt.Errorf("parse xml result: %w", err)
	}
	logger.InfoContext(ctx, "xf ise parse xml result successfully")
	return a.convertToResult(resultXML, result), nil
}

func exceptInfoString(code int) string {
	if code == 0 {
		return ""
	}
	return strconv.Itoa(code)
}

// Close 关闭客户端
func (a *XFEvaluationAdapter) Close() error {
	return nil
}

// convertToResult 将 speechAssessResult 转为领域层 EvaluationResult（展平句内单词供业务使用）。
func (a *XFEvaluationAdapter) convertToResult(xmlBytes []byte, sdk *speechAssessResult) *domain.EvaluationResult {
	return &domain.EvaluationResult{
		TotalScore:     sdk.TotalScore,
		AccuracyScore:  sdk.AccuracyScore,
		FluencyScore:   sdk.FluencyScore,
		IntegrityScore: sdk.IntegrityScore,
		StandardScore:  sdk.StandardScore,
		Words:          flattenWordsForDomain(sdk),
		RawXML:         string(xmlBytes),
		IsRejected:     sdk.IsRejected,
		ExceptInfo:     exceptInfoString(sdk.ExceptInfo),
	}
}

// flattenWordsForDomain 汇总所有句子中的词（跳过静音/噪声标记），并附带音素明细。
func flattenWordsForDomain(sdk *speechAssessResult) []domain.WordEvaluationResult {
	var words []domain.WordEvaluationResult
	for _, sent := range sdk.Sentences {
		for _, w := range sent.Words {
			if w.Content == "sil" || w.Content == "silv" || w.Content == "fil" {
				continue
			}
			words = append(words, domain.WordEvaluationResult{
				Word:      w.Content,
				Score:     w.TotalScore,
				BeginTime: w.BegPos,
				EndTime:   w.EndPos,
				DpMessage: w.DpMessage,
				Phonemes:  phonesToDomain(w),
			})
		}
	}
	return words
}

func phonesToDomain(w wordResult) []domain.PhonemeEvaluationResult {
	var out []domain.PhonemeEvaluationResult
	for _, sy := range w.Sylls {
		for _, ph := range sy.Phones {
			if ph.Content == "sil" || ph.Content == "fil" {
				continue
			}
			out = append(out, domain.PhonemeEvaluationResult{
				Phoneme:   ph.Content,
				Score:     0,
				BeginTime: ph.BegPos,
				EndTime:   ph.EndPos,
			})
		}
	}
	return out
}

// buildAuthURL 构建带认证的 WebSocket URL
func (a *XFEvaluationAdapter) buildAuthURL() (string, error) {
	// 生成 RFC1123 格式的时间戳
	date := time.Now().UTC().Format(time.RFC1123)

	// 构建签名原文：host + date + request-line
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", iseHost, date, isePath)

	// HMAC-SHA256 签名
	mac := hmac.New(sha256.New, []byte(a.apiSecret))
	mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 构建 authorization
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		a.apiKey, signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	// 构建最终 URL
	wsURL := fmt.Sprintf("wss://%s%s?authorization=%s&date=%s&host=%s",
		iseHost, isePath,
		url.QueryEscape(authorization),
		url.QueryEscape(date),
		url.QueryEscape(iseHost))

	return wsURL, nil
}

// sendSSBFrame 发送参数上传帧（第一阶段）
func (a *XFEvaluationAdapter) sendSSBFrame(ctx context.Context, conn *websocket.Conn, req *speechAssessRequest) error {
	// 文本需要加 UTF-8 BOM 头
	text := "\uFEFF"
	// 针对不同的category，text需要做不同的处理
	switch req.Category {
	case "read_sentence":
		text = "\uFEFF" + "[content]\n" + req.Text
	case "read_word":
		text = "\uFEFF" + "[word]\n" + req.Text
	case "read_chapter":
		text = "\uFEFF" + "[content]\n" + req.Text
	}

	frame := webSocketFrame{
		Common: &commonParams{
			AppID: a.appID,
		},
		Business: &businessParams{
			Sub:      "ise",
			Ent:      "en_vip",
			Category: req.Category,
			Cmd:      "ssb",
			Text:     text,
			Tte:      "utf-8",
			TtpSkip:  true,
			Aue:      "raw",
			Auf:      "audio/L16;rate=16000",
			Rstcd:    "utf8",
		},
		Data: &dataParams{
			Status: 0,
		},
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal ssb frame: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write ssb frame: %w", err)
	}

	logger.DebugContext(ctx, "xf ise: ssb frame sent", "text", req.Text, "category", req.Category)
	return nil
}

// sendAudioFrames 分片发送音频数据
func (a *XFEvaluationAdapter) sendAudioFrames(ctx context.Context, conn *websocket.Conn, audioData []byte) error {
	totalLen := len(audioData)
	if totalLen == 0 {
		// 没有音频数据，直接发送最后一帧
		return a.sendAudioChunk(ctx, conn, nil, 4, 2)
	}

	offset := 0
	isFirst := true

	for offset < totalLen {
		end := offset + audioChunkSize
		if end > totalLen {
			end = totalLen
		}
		chunk := audioData[offset:end]
		isLast := end >= totalLen

		var aus int
		var status int

		switch {
		case isFirst:
			aus = 1 // 第一帧音频
			status = 1
			isFirst = false
		case isLast:
			aus = 4 // 最后一帧音频
			status = 2
		default:
			aus = 2 // 中间帧音频
			status = 1
		}

		if err := a.sendAudioChunk(ctx, conn, chunk, aus, status); err != nil {
			return fmt.Errorf("send chunk at offset %d: %w", offset, err)
		}

		offset = end

		// 每帧之间间隔 40ms，与 Node.js Demo 一致
		if !isLast {
			time.Sleep(audioSendInterval)
		}
	}

	logger.DebugContext(ctx, "xf ise: audio frames sent", "total_bytes", totalLen,
		"chunks", (totalLen+audioChunkSize-1)/audioChunkSize)
	return nil
}

// sendAudioChunk 发送单个音频分片
func (a *XFEvaluationAdapter) sendAudioChunk(ctx context.Context, conn *websocket.Conn, chunk []byte, aus int, status int) error {
	frame := webSocketFrame{
		Common: &commonParams{
			AppID: a.appID,
		},
		Business: &businessParams{
			Cmd: "auw",
			Aus: aus,
			Aue: "raw",
		},
		Data: &dataParams{
			Status: status,
			Data:   base64.StdEncoding.EncodeToString(chunk),
		},
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal audio chunk: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write audio chunk: %w", err)
	}
	return nil
}

// receiveResult 接收评测结果，循环读取直到 status==2
func (a *XFEvaluationAdapter) receiveResult(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

		_, message, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read message: %w", err)
		}

		var resp responseFrame
		if err := json.Unmarshal(message, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		// 检查错误码
		if resp.Code != 0 {
			return nil, NewError(resp.Code, resp.Message)
		}

		if resp.Data == nil {
			continue
		}

		// status==2 表示最终结果
		if resp.Data.Status == 2 {
			if resp.Data.Data == "" {
				return nil, fmt.Errorf("final response has empty data")
			}
			// Base64 解码获取 XML
			xmlBytes, err := base64.StdEncoding.DecodeString(resp.Data.Data)
			if err != nil {
				return nil, fmt.Errorf("base64 decode result: %w", err)
			}
			logger.InfoContext(ctx, "xf ise: final result received", "sid", resp.Sid)
			return xmlBytes, nil
		}
	}
}

// parseXMLResult 解析 XML 评测结果
func (a *XFEvaluationAdapter) parseXMLResult(xmlBytes []byte) (*speechAssessResult, error) {
	var result xmlResult
	if err := xml.Unmarshal(xmlBytes, &result); err != nil {
		return nil, fmt.Errorf("xml unmarshal: %w", err)
	}
	var block *xmlReadBlock
	switch {
	case result.ReadSentence != nil:
		block = result.ReadSentence
	case result.ReadWord != nil:
		block = result.ReadWord
	case result.ReadChapter != nil:
		block = result.ReadChapter
	default:
		return nil, fmt.Errorf("no matching read block found in xml result")
	}
	if block.RecPaper == nil {
		return nil, fmt.Errorf("rec_paper is nil")
	}
	var item *xmlReadItem
	switch {
	case block.RecPaper.ReadSentence != nil:
		item = block.RecPaper.ReadSentence
	case block.RecPaper.ReadWord != nil:
		item = block.RecPaper.ReadWord
	case block.RecPaper.ReadChapter != nil:
		item = block.RecPaper.ReadChapter
	default:
		return nil, fmt.Errorf("no matching read item in rec_paper")
	}

	assess := &speechAssessResult{
		RawXML:         string(xmlBytes),
		TotalScore:     parseFloat(item.TotalScore),
		AccuracyScore:  parseFloat(item.AccuracyScore),
		FluencyScore:   parseFloat(item.FluencyScore),
		IntegrityScore: parseFloat(item.IntegrityScore),
		StandardScore:  parseFloat(item.StandardScore),
		IsRejected:     item.IsRejected == "true",
		ExceptInfo:     parseInt(item.ExceptInfo),
		BegPos:         parseInt(item.BegPos),
		EndPos:         parseInt(item.EndPos),
		Content:        item.Content,
		WordCount:      parseInt(item.WordCount),
		Sentences:      mapXMLSentences(item.Sentences),
	}
	return assess, nil
}

func mapXMLSentences(xs []xmlSentence) []sentenceResult {
	if len(xs) == 0 {
		return nil
	}
	out := make([]sentenceResult, 0, len(xs))
	for _, s := range xs {
		out = append(out, sentenceResult{
			AccuracyScore: parseFloat(s.AccuracyScore),
			StandardScore: parseFloat(s.StandardScore),
			FluencyScore:  parseFloat(s.FluencyScore),
			TotalScore:    parseFloat(s.TotalScore),
			BegPos:        parseInt(s.BegPos),
			EndPos:        parseInt(s.EndPos),
			Content:       s.Content,
			WordCount:     parseInt(s.WordCount),
			Words:         mapXMLWords(s.Words),
		})
	}
	return out
}

func mapXMLWords(xws []xmlWord) []wordResult {
	if len(xws) == 0 {
		return nil
	}
	out := make([]wordResult, 0, len(xws))
	for _, w := range xws {
		out = append(out, wordResult{
			Content:    w.Content,
			BegPos:     parseInt(w.BegPos),
			EndPos:     parseInt(w.EndPos),
			DpMessage:  parseInt(w.DpMessage),
			TotalScore: parseFloat(w.TotalScore),
			WerrMsg:    w.WerrMsg,
			Sylls:      mapXMLSylls(w.Sylls),
		})
	}
	return out
}

func mapXMLSylls(xss []xmlSyll) []syllResult {
	if len(xss) == 0 {
		return nil
	}
	out := make([]syllResult, 0, len(xss))
	for _, s := range xss {
		out = append(out, syllResult{
			Content:    s.Content,
			BegPos:     parseInt(s.BegPos),
			EndPos:     parseInt(s.EndPos),
			SerrMsg:    parseInt(s.SerrMsg),
			SyllAccent: parseInt(s.SyllAccent),
			Phones:     mapXMLPhones(s.Phones),
		})
	}
	return out
}

func mapXMLPhones(xps []xmlPhone) []phoneResult {
	if len(xps) == 0 {
		return nil
	}
	out := make([]phoneResult, 0, len(xps))
	for _, p := range xps {
		out = append(out, phoneResult{
			Content:   p.Content,
			BegPos:    parseInt(p.BegPos),
			EndPos:    parseInt(p.EndPos),
			DpMessage: parseInt(p.DpMessage),
		})
	}
	return out
}
