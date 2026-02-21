// Package xf 提供科大讯飞语音评测的具体实现
// 此包内的所有类型对外部（Service 层）不可见，
// 仅通过 XFEvaluationAdapter 实现 domain.EvaluationProvider 接口对外暴露
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
	"strconv"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	// iseHostURL 讯飞 ISE WebSocket 地址
	iseHost = "ise-api.xfyun.cn"
	isePath = "/v2/open-ise"

	// audioChunkSize 音频分片大小 (字节)，与 Node.js Demo 一致
	audioChunkSize = 1280

	// audioSendInterval 每片音频发送间隔
	audioSendInterval = 40 * time.Millisecond

	// wsReadDeadline WebSocket 读取超时
	wsReadDeadline = 30 * time.Second
)

// internalClient 科大讯飞内部 SDK 客户端
type internalClient struct {
	appID     string
	apiKey    string
	apiSecret string
}

// newInternalClient 创建内部客户端
func newInternalClient(cfg config.XunFeiConfig) *internalClient {
	return &internalClient{
		appID:     cfg.AppID,
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
	}
}

// speechAssess 调用讯飞语音评测 API
// 流程：建连 → 发送 SSB 参数帧 → 分片发送音频 → 接收结果 → 解析 XML
func (c *internalClient) speechAssess(ctx context.Context, req *speechAssessRequest) (*speechAssessResult, error) {
	// 1. 构建鉴权 URL 并建立 WebSocket 连接
	wsURL, err := c.buildAuthURL()
	if err != nil {
		return nil, fmt.Errorf("build auth url: %w", err)
	}

	logger.DebugContext(ctx, "xf ise: connecting", "url_length", len(wsURL))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	logger.DebugContext(ctx, "xf ise: connected")

	// 2. 发送 SSB 参数帧（cmd=ssb, data.status=0）
	if err := c.sendSSBFrame(ctx, conn, req); err != nil {
		return nil, fmt.Errorf("send ssb frame: %w", err)
	}

	// 3. 分片发送音频（cmd=auw）
	if err := c.sendAudioFrames(ctx, conn, req.AudioData); err != nil {
		return nil, fmt.Errorf("send audio frames: %w", err)
	}

	logger.DebugContext(ctx, "xf ise: all frames sent, waiting for result")

	// 4. 接收评测结果
	resultXML, err := c.receiveResult(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("receive result: %w", err)
	}

	logger.DebugContext(ctx, "xf ise: received result", "xml_length", len(resultXML))

	// 5. 解析 XML 评测结果
	result, err := parseXMLResult(resultXML, req.Category)
	if err != nil {
		return nil, fmt.Errorf("parse xml result: %w", err)
	}

	return result, nil
}

// sendSSBFrame 发送参数上传帧（第一阶段）
func (c *internalClient) sendSSBFrame(ctx context.Context, conn *websocket.Conn, req *speechAssessRequest) error {
	// 文本需要加 UTF-8 BOM 头
	text := "\uFEFF" + req.Text

	frame := webSocketFrame{
		Common: &commonParams{
			AppID: c.appID,
		},
		Business: &businessParams{
			Sub:      "ise",
			Ent:      req.Language,
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
func (c *internalClient) sendAudioFrames(ctx context.Context, conn *websocket.Conn, audioData []byte) error {
	totalLen := len(audioData)
	if totalLen == 0 {
		// 没有音频数据，直接发送最后一帧
		return c.sendAudioChunk(ctx, conn, nil, 4, 2)
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

		if err := c.sendAudioChunk(ctx, conn, chunk, aus, status); err != nil {
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
func (c *internalClient) sendAudioChunk(ctx context.Context, conn *websocket.Conn, chunk []byte, aus int, status int) error {
	frame := webSocketFrame{
		Common: &commonParams{
			AppID: c.appID,
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
func (c *internalClient) receiveResult(ctx context.Context, conn *websocket.Conn) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

		_, message, err := conn.ReadMessage()
		if err != nil {
			return "", fmt.Errorf("read message: %w", err)
		}

		var resp responseFrame
		if err := json.Unmarshal(message, &resp); err != nil {
			return "", fmt.Errorf("unmarshal response: %w", err)
		}

		// 检查错误码
		if resp.Code != 0 {
			return "", NewError(resp.Code, resp.Message)
		}

		if resp.Data == nil {
			continue
		}

		// status==2 表示最终结果
		if resp.Data.Status == 2 {
			if resp.Data.Data == "" {
				return "", fmt.Errorf("final response has empty data")
			}

			// Base64 解码获取 XML
			xmlBytes, err := base64.StdEncoding.DecodeString(resp.Data.Data)
			if err != nil {
				return "", fmt.Errorf("base64 decode result: %w", err)
			}

			logger.DebugContext(ctx, "xf ise: final result received", "sid", resp.Sid)
			return string(xmlBytes), nil
		}
	}
}

// parseXMLResult 解析 XML 评测结果为内部结构
func parseXMLResult(xmlStr string, category string) (*speechAssessResult, error) {
	var result xmlResult
	if err := xml.Unmarshal([]byte(xmlStr), &result); err != nil {
		return nil, fmt.Errorf("xml unmarshal: %w", err)
	}

	// 根据题型取对应的 block
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

	// 取对应题型的评测项
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

	assessResult := &speechAssessResult{
		TotalScore:   parseFloat(item.TotalScore),
		Accuracy:     parseFloat(item.AccuracyScore),
		Fluency:      parseFloat(item.FluencyScore),
		Completeness: parseFloat(item.IntegrityScore),
		Intonation:   parseFloat(item.StandardScore),
	}

	// 解析单词级结果
	for _, sentence := range item.Sentences {
		for _, word := range sentence.Words {
			w := wordResult{
				Word:      word.Content,
				Score:     parseFloat(word.TotalScore),
				BeginTime: parseInt(word.BegPos),
				EndTime:   parseInt(word.EndPos),
				DpMessage: parseInt(word.DpMessage),
			}

			// 解析音素级结果（从音节下提取音素）
			for _, syll := range word.Sylls {
				for _, phone := range syll.Phones {
					// 跳过 sil/fil 等非语音音素
					if phone.RecNodeType == "sil" || phone.RecNodeType == "fil" {
						continue
					}
					p := phonemeResult{
						Phoneme:   phone.Content,
						BeginTime: parseInt(phone.BegPos),
						EndTime:   parseInt(phone.EndPos),
					}
					w.Phonemes = append(w.Phonemes, p)
				}
			}

			assessResult.Words = append(assessResult.Words, w)
		}
	}

	return assessResult, nil
}

// buildAuthURL 构建带认证的 WebSocket URL
func (c *internalClient) buildAuthURL() (string, error) {
	// 生成 RFC1123 格式的时间戳
	date := time.Now().UTC().Format(time.RFC1123)

	// 构建签名原文：host + date + request-line
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", iseHost, date, isePath)

	// HMAC-SHA256 签名
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 构建 authorization
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		c.apiKey, signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	// 构建最终 URL
	wsURL := fmt.Sprintf("wss://%s%s?authorization=%s&date=%s&host=%s",
		iseHost, isePath,
		url.QueryEscape(authorization),
		url.QueryEscape(date),
		url.QueryEscape(iseHost))

	return wsURL, nil
}

// close 关闭客户端
func (c *internalClient) close() error {
	return nil
}

// ===================== 辅助函数 =====================

// parseFloat 安全解析浮点数
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseInt 安全解析整数
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}
