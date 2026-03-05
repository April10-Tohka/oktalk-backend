// Package domain 定义微信客户端领域接口
package domain

// WechatOAuthToken 微信 OAuth Token 响应
type WechatOAuthToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid,omitempty"`
}

// WechatUserInfo 微信用户信息
type WechatUserInfo struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid,omitempty"`
}

// WechatClient 微信客户端接口
type WechatClient interface {
	// GetOAuthToken 用 auth_code 换取 openid 和 access_token
	GetOAuthToken(authCode string) (*WechatOAuthToken, error)
	// GetUserInfo 获取微信用户信息（昵称、头像）
	GetUserInfo(accessToken, openID string) (*WechatUserInfo, error)
}
