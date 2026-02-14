// Package xf 提供科大讯飞语音评测的具体实现
// 此包内的所有类型对外部（Service 层）不可见，
// 仅通过 XFSpeechAdapter 实现 domain.SpeechAssessor 接口对外暴露
package xf

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"pronunciation-correction-system/internal/config"
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
func (c *internalClient) speechAssess(ctx context.Context, req *speechAssessRequest) (*speechAssessResponse, error) {
	// TODO: 实现科大讯飞语音评测 API 调用
	// 1. 构建 WebSocket 连接（使用 buildAuthURL）
	// 2. 发送音频数据
	// 3. 接收评测结果
	// 4. 解析并返回结果
	_ = ctx
	_ = req
	return nil, fmt.Errorf("xf speech assess not implemented yet")
}

// buildAuthURL 构建带认证的 WebSocket URL
func (c *internalClient) buildAuthURL() (string, error) {
	host := "ise-api.xfyun.cn"
	path := "/v2/open-ise"

	// 生成 RFC1123 格式的时间戳
	date := time.Now().UTC().Format(time.RFC1123)

	// 构建签名原文
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", host, date, path)

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
		host, path,
		url.QueryEscape(authorization),
		url.QueryEscape(date),
		url.QueryEscape(host))

	return wsURL, nil
}

// close 关闭客户端
func (c *internalClient) close() error {
	// TODO: 清理资源
	return nil
}

// --- 内部 SDK 数据结构 ---

// speechAssessRequest 语音评测请求
type speechAssessRequest struct {
	Text      string `json:"text"`
	AudioData []byte `json:"audio_data"`
	AudioURL  string `json:"audio_url"`
	Category  string `json:"category"`
	Language  string `json:"language"`
}

// speechAssessResponse 语音评测响应
type speechAssessResponse struct {
	RequestID    string              `json:"request_id"`
	Status       int                 `json:"status"`
	Result       *speechAssessResult `json:"result"`
	ErrorCode    int                 `json:"error_code"`
	ErrorMessage string              `json:"error_message"`
}

// speechAssessResult 语音评测结果
type speechAssessResult struct {
	TotalScore   float64      `json:"total_score"`
	Accuracy     float64      `json:"accuracy"`
	Fluency      float64      `json:"fluency"`
	Completeness float64      `json:"completeness"`
	Intonation   float64      `json:"intonation"`
	Words        []wordResult `json:"words"`
}

// wordResult 单词评测结果
type wordResult struct {
	Word      string          `json:"word"`
	Score     float64         `json:"score"`
	BeginTime int             `json:"begin_time"`
	EndTime   int             `json:"end_time"`
	Phonemes  []phonemeResult `json:"phonemes"`
}

// phonemeResult 音素评测结果
type phonemeResult struct {
	Phoneme   string  `json:"phoneme"`
	Score     float64 `json:"score"`
	BeginTime int     `json:"begin_time"`
	EndTime   int     `json:"end_time"`
}
