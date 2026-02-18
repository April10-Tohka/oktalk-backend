// Package handler 提供用户信息 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/service"
)

// UserHandler 用户信息处理器
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建 UserHandler
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile GET /api/v1/user/profile
// 获取当前登录用户的详细信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 从 Context 获取 user_id
	// 2. 调用 h.userService.GetProfile(ctx, userID)
	// 3. 成功：OK(c, profileResponse)
	// 4. 失败：NotFound / InternalError
	InternalError(c, "not implemented")
}

// UpdateProfile PUT /api/v1/user/profile
// 更新当前登录用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// TODO: Step2 实现
	// 1. 解析 multipart/form-data 或 JSON: username, avatar(file), bio, grade
	// 2. 从 Context 获取 user_id
	// 3. 调用 h.userService.UpdateProfile(ctx, req)
	// 4. 成功：OK(c, gin.H{"message": "profile updated"})
	// 5. 失败：BadRequest / InternalError
	InternalError(c, "not implemented")
}
