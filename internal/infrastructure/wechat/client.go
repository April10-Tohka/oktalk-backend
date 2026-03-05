// Package wechat 提供微信开放平台客户端实现
package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// wechatOAuthResp 微信 OAuth 接口原始响应（包含 errcode/errmsg）
type wechatOAuthResp struct {
	domain.WechatOAuthToken
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// wechatUserInfoResp 微信用户信息接口原始响应
type wechatUserInfoResp struct {
	domain.WechatUserInfo
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// Client 微信客户端实现
type Client struct {
	appID     string
	appSecret string
	httpClient *http.Client
}

// NewClient 创建微信客户端
func NewClient(cfg config.WeChatConfig) domain.WechatClient {
	return &Client{
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOAuthToken 用 auth_code 换取 openid 和 access_token
func (c *Client) GetOAuthToken(authCode string) (*domain.WechatOAuthToken, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.appID, c.appSecret, authCode,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		slog.Error("wechat oauth request failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("微信授权请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取微信响应失败: %w", err)
	}

	var result wechatOAuthResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析微信响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		slog.Error("wechat oauth error",
			slog.Int("errcode", result.ErrCode),
			slog.String("errmsg", result.ErrMsg),
		)
		return nil, fmt.Errorf("微信授权失败(errcode=%d): %s", result.ErrCode, result.ErrMsg)
	}

	return &result.WechatOAuthToken, nil
}

// GetUserInfo 获取微信用户信息（昵称、头像）
func (c *Client) GetUserInfo(accessToken, openID string) (*domain.WechatUserInfo, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		accessToken, openID,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		slog.Error("wechat userinfo request failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("获取微信用户信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取微信响应失败: %w", err)
	}

	var result wechatUserInfoResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析微信用户信息失败: %w", err)
	}

	if result.ErrCode != 0 {
		slog.Warn("wechat userinfo error",
			slog.Int("errcode", result.ErrCode),
			slog.String("errmsg", result.ErrMsg),
		)
		return nil, fmt.Errorf("获取微信用户信息失败(errcode=%d): %s", result.ErrCode, result.ErrMsg)
	}

	return &result.WechatUserInfo, nil
}
