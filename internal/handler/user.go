// Package handler 提供用户相关的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/service/user"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService user.Service
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService user.Service) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile 获取用户信息
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	// TODO: 从上下文获取用户 ID
	// TODO: 调用服务获取用户信息

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// UpdateProfile 更新用户信息
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// TODO: 解析请求参数
	// TODO: 调用服务更新用户信息

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// GetLearningStats 获取学习统计
// GET /api/v1/user/stats
func (h *UserHandler) GetLearningStats(c *gin.Context) {
	// TODO: 从上下文获取用户 ID
	// TODO: 调用服务获取学习统计

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *UserHandler) Register(c *gin.Context) {
	// TODO: 解析注册参数
	// TODO: 调用服务创建用户

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *UserHandler) Login(c *gin.Context) {
	// TODO: 解析登录参数
	// TODO: 调用服务验证用户并生成 Token

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *UserHandler) Logout(c *gin.Context) {
	// TODO: 使当前 Token 失效

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}
