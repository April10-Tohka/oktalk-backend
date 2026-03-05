// Package sms 提供短信服务客户端
package sms

import (
	"fmt"
	"log/slog"

	"pronunciation-correction-system/internal/config"
)

// Client 短信客户端接口
type Client interface {
	// Send 发送短信验证码
	Send(phone, code string) error
}

// mockClient Mock 短信客户端（开发调试用，仅打印日志）
type mockClient struct {
	signName     string
	templateCode string
}

// NewClient 根据配置创建短信客户端
// 当前仅实现 mock 模式，后续可扩展阿里云/腾讯云等
func NewClient(cfg config.SMSConfig) Client {
	switch cfg.ActiveProvider {
	case "mock":
		return &mockClient{
			signName:     cfg.SignName,
			templateCode: cfg.TemplateCode,
		}
	default:
		// 默认使用 mock
		return &mockClient{
			signName:     cfg.SignName,
			templateCode: cfg.TemplateCode,
		}
	}
}

// Send 发送短信验证码（Mock: 仅打印日志）
func (c *mockClient) Send(phone, code string) error {
	slog.Info(fmt.Sprintf("[SMS Mock] 发送验证码: phone=%s, code=%s, sign=%s, template=%s",
		phone, code, c.signName, c.templateCode))
	return nil
}
