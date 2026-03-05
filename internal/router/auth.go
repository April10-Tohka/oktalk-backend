// Package router 提供认证相关路由
package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

// setupAuthRoutes 注册认证路由（无需登录 - 公开接口）
func setupAuthRoutes(rg *gin.RouterGroup, h *handler.AuthHandler) {
	auth := rg.Group("/auth")
	{
		// 短信验证码
		sms := auth.Group("/sms")
		{
			sms.POST("/send", h.SendSMS)   // 发送短信验证码
			sms.POST("/login", h.SMSLogin)  // 手机验证码登录/自动注册
		}

		// 微信登录
		wechat := auth.Group("/wechat")
		{
			wechat.POST("/login", h.WechatLogin) // 微信 App SSO 登录/自动注册
		}

		// Token 相关（无需认证）
		token := auth.Group("/token")
		{
			token.POST("/refresh", h.RefreshToken) // 刷新 Access Token
		}
	}
}

// setupAuthProtectedRoutes 注册需要认证的 auth 路由
func setupAuthProtectedRoutes(rg *gin.RouterGroup, h *handler.AuthHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/logout", h.Logout) // 退出登录（需认证）
	}
}
