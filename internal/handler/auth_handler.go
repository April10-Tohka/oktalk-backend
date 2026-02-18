// Package handler 提供认证 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"

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

// Login POST /api/v1/auth/login
// 用户登录，返回 JWT Token
func (h *AuthHandler) Login(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 JSON 请求体: email, password
	// 2. 调用 h.authService.Login(ctx, email, password)
	// 3. 成功：OK(c, authResponse)
	// 4. 失败：Unauthorized / BadRequest
	InternalError(c, "not implemented")
}

// Register POST /api/v1/auth/register
// 用户注册，注册后自动登录返回 Token
func (h *AuthHandler) Register(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 JSON 请求体: email, password, username
	// 2. 调用 h.authService.Register(ctx, req)
	// 3. 成功：OK(c, authResponse)
	// 4. 失败：BadRequest(c, "email already registered") / InternalError
	InternalError(c, "not implemented")
}

// Logout POST /api/v1/auth/logout
// 用户登出，使当前 Token 失效
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 从 Context 获取 user_id 和 token
	// 2. 调用 h.authService.Logout(ctx, userID, token)
	// 3. 成功：OK(c, gin.H{"message": "logged out"})
	// 4. 失败：InternalError
	InternalError(c, "not implemented")
}

// RefreshToken POST /api/v1/auth/refresh
// 刷新 JWT Token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 JSON 请求体: refresh_token
	// 2. 调用 h.authService.RefreshToken(ctx, refreshToken)
	// 3. 成功：OK(c, tokenResponse)
	// 4. 失败：Unauthorized / InternalError
	InternalError(c, "not implemented")
}
