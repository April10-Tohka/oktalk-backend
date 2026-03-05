// Package handler 提供认证 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler/middleware"
	"pronunciation-correction-system/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SendSMS POST /api/v1/auth/sms/send
// 发送短信验证码
func (h *AuthHandler) SendSMS(c *gin.Context) {
	var req service.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 注入 IP
	req.IP = c.ClientIP()

	resp, err := h.authService.SendSMS(c.Request.Context(), &req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}

// SMSLogin POST /api/v1/auth/sms/login
// 手机验证码登录/自动注册
func (h *AuthHandler) SMSLogin(c *gin.Context) {
	var req service.SMSLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 注入 IP 和 User-Agent
	req.IP = c.ClientIP()

	resp, err := h.authService.SMSLogin(c.Request.Context(), &req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}

// WechatLogin POST /api/v1/auth/wechat/login
// 微信 App SSO 登录/自动注册
func (h *AuthHandler) WechatLogin(c *gin.Context) {
	var req service.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 注入 IP
	req.IP = c.ClientIP()

	resp, err := h.authService.WechatLogin(c.Request.Context(), &req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}

// RefreshToken POST /api/v1/auth/token/refresh
// 刷新 Access Token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req service.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}

// Logout POST /api/v1/auth/logout
// 退出登录（需要认证）
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get(string(middleware.UserIDKey))
	jti, _ := c.Get(string(middleware.JTIKey))

	req := &service.LogoutRequest{
		UserID: userID.(string),
		JTI:    jti.(string),
		IP:     c.ClientIP(),
	}

	err := h.authService.Logout(c.Request.Context(), req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, gin.H{"message": "已退出登录"})
}

// handleAuthError 统一处理认证业务错误
func handleAuthError(c *gin.Context, err error) {
	if authErr, ok := err.(*service.AuthError); ok {
		switch authErr.Code {
		case 400:
			Fail(c, http.StatusBadRequest, authErr.Code, authErr.Message)
		case 401:
			Fail(c, http.StatusUnauthorized, authErr.Code, authErr.Message)
		case 403:
			Fail(c, http.StatusForbidden, authErr.Code, authErr.Message)
		case 404:
			Fail(c, http.StatusNotFound, authErr.Code, authErr.Message)
		case 409:
			Fail(c, http.StatusConflict, authErr.Code, authErr.Message)
		case 429:
			Fail(c, http.StatusTooManyRequests, authErr.Code, authErr.Message)
		default:
			InternalError(c, authErr.Message)
		}
		return
	}
	InternalError(c, err.Error())
}
