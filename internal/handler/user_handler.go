// Package handler 提供用户信息 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler/middleware"
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
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	resp, err := h.userService.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}

// UpdateProfile PUT /api/v1/user/profile
// 更新当前登录用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	req.UserID = userID.(string)

	resp, err := h.userService.UpdateProfile(c.Request.Context(), &req)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	OK(c, resp)
}
