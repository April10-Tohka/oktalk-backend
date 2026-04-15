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
	Send(phone, code string) error
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
func (c *aliyunClient) Send(phone, code string) error {

	sendSmsVerifyCodeRequest := &dypnsapi20170525.SendSmsVerifyCodeRequest{
		SignName:      tea.String(c.signName),
		TemplateCode:  tea.String(c.templateCode),
		PhoneNumber:   tea.String(phone),
		TemplateParam: tea.String("{\"code\":\"##code##\",\"min\":\"5\"}"),
	}

	resp, err := c.client.SendSmsVerifyCode(sendSmsVerifyCodeRequest)
	if err != nil {
		return fmt.Errorf("aliyun sms send failed: %w", err)
	}

	if resp.Body == nil || tea.StringValue(resp.Body.Code) != "OK" {
		msg := "unknown error"
		if resp.Body != nil {
			msg = tea.StringValue(resp.Body.Message)
		}
		return fmt.Errorf("aliyun sms send failed: message=%s", msg)
	}

	slog.Info("Aliyun SMS sent successfully", "phone", phone, "requestId", tea.StringValue(resp.Body.RequestId))
	return nil
}
