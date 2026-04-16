// Package sms 提供短信服务客户端
package sms

import (
	"fmt"
	"log/slog"

	"pronunciation-correction-system/internal/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypnsapi20170525 "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
)

// Client 短信客户端接口
type Client interface {
	// Send 发送短信验证码
	Send(phone, code string) (*SendResult, error)
	// Verify 验证短信验证码
	Verify(phone, code string) (*VerifyResult, error)
}

// SendResult 发送结果，包含阿里云返回的业务ID和明文验证码
type SendResult struct {
	BizId      string // 业务ID，可用于审计
	VerifyCode string // 明文验证码，供 Service 层存入 Redis
	RequestId  string // 请求ID
}

// VerifyResult 定义短信验证的结果
type VerifyResult struct {
	Success bool   // 系统调用是否成功
	Pass    bool   // 验证码是否正确
	Message string // 附加信息（如：验证码过期、次数超限等）
}

// aliyunClient 阿里云短信客户端
type aliyunClient struct {
	client       *dypnsapi20170525.Client
	signName     string
	templateCode string
}

// NewClient 根据配置创建短信客户端
func NewClient(cfg config.SMSConfig) Client {
	switch cfg.ActiveProvider {
	case "aliyun":
		client, err := createAliyunClient(cfg.Aliyun)
		if err != nil {
			slog.Error("Failed to create aliyun sms client", "error", err)
		}
		return &aliyunClient{
			client:       client,
			signName:     cfg.Aliyun.SignName,
			templateCode: cfg.Aliyun.TemplateCode,
		}
	default:
		return nil
	}
}

// createAliyunClient 创建阿里云短信客户端
func createAliyunClient(cfg config.AliyunSMSConfig) (*dypnsapi20170525.Client, error) {

	credentialsConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(cfg.AccessKeyID).
		SetAccessKeySecret(cfg.AccessKeySecret)
	akCredential, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return nil, err
	}
	config := &openapi.Config{
		Credential: akCredential,
		Endpoint:   tea.String(cfg.Endpoint),
	}

	return dypnsapi20170525.NewClient(config)

}

// Send 发送短信验证码（阿里云）
func (c *aliyunClient) Send(phone, code string) (*SendResult, error) {

	req := &dypnsapi20170525.SendSmsVerifyCodeRequest{
		SignName:         tea.String(c.signName),
		TemplateCode:     tea.String(c.templateCode),
		PhoneNumber:      tea.String(phone),
		TemplateParam:    tea.String("{\"code\":\"##code##\",\"min\":\"5\"}"),
		ReturnVerifyCode: tea.Bool(true),
	}

	resp, err := c.client.SendSmsVerifyCode(req)
	if err != nil {
		slog.Error("Aliyun SMS API call failed", "phone", phone, "err", err)
		return nil, fmt.Errorf("sms service unavailable")
	}

	if resp.Body == nil || tea.StringValue(resp.Body.Code) != "OK" {
		msg := "unknown error"
		if resp.Body != nil {
			msg = tea.StringValue(resp.Body.Message)
		}
		slog.Warn("Aliyun SMS send failed", "phone", phone, "msg", msg)
		return nil, fmt.Errorf("send failed: %s", msg)
	}

	// 提取关键数据
	bizId := ""
	verifyCode := ""
	requestId := tea.StringValue(resp.Body.RequestId)

	if resp.Body.Model != nil {
		bizId = tea.StringValue(resp.Body.Model.BizId)
		verifyCode = tea.StringValue(resp.Body.Model.VerifyCode)
	}

	if verifyCode == "" {
		return nil, fmt.Errorf("aliyun returned empty verify code")
	}

	slog.Info("Aliyun SMS sent", "phone", phone, "bizId", bizId)

	return &SendResult{
		BizId:      bizId,
		VerifyCode: verifyCode,
		RequestId:  requestId,
	}, nil
}

// Verify 验证短信验证码
// 返回:
// - result: 验证结果对象
// - err: 仅当发生系统级错误（网络、鉴权、参数非法）时返回
func (c *aliyunClient) Verify(phone, code string) (*VerifyResult, error) {
	req := &dypnsapi20170525.CheckSmsVerifyCodeRequest{
		PhoneNumber: tea.String(phone),
		VerifyCode:  tea.String(code),
	}

	resp, err := c.client.CheckSmsVerifyCode(req)
	if err != nil {
		slog.Error("Aliyun SMS verify API call failed", "phone", phone, "err", err)
		return nil, fmt.Errorf("aliyun sms verify api error: %w", err)
	}

	// 1. 检查 HTTP/API 层面的响应状态
	if resp.Body == nil {
		return nil, fmt.Errorf("aliyun sms verify response body is nil")
	}

	apiCode := tea.StringValue(resp.Body.Code)
	if apiCode != "OK" {
		msg := tea.StringValue(resp.Body.Message)
		slog.Warn("Aliyun SMS verify API returned non-OK code", "code", apiCode, "msg", msg)
		// 这里可以根据具体 Code 做更细致的处理，比如 ISV.BUSINESS_LIMIT_CONTROL
		return &VerifyResult{
			Success: false,
			Pass:    false,
			Message: fmt.Sprintf("API Error: %s", msg),
		}, nil // 注意：这里返回 nil error，因为这是业务可预期的失败
	}

	// 2. 检查业务层面的验证结果
	model := resp.Body.Model
	if model == nil {
		return nil, fmt.Errorf("aliyun sms verify model is nil")
	}

	verifyResult := tea.StringValue(model.VerifyResult)

	slog.Info("Aliyun SMS verify result", "phone", phone, "result", verifyResult)

	switch verifyResult {
	case "PASS":
		return &VerifyResult{
			Success: true,
			Pass:    true,
			Message: "Verification successful",
		}, nil
	case "UNKNOWN":
		// 验证码错误、过期、不匹配等
		return &VerifyResult{
			Success: true,
			Pass:    false,
			Message: fmt.Sprintf("Unknown verify status: %s", verifyResult),
		}, nil
	default:
		// 未知状态
		return &VerifyResult{
			Success: true,
			Pass:    false,
			Message: fmt.Sprintf("Unknown verify status: %s", verifyResult),
		}, nil
	}
}
