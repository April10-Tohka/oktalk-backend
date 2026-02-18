// Package service 提供用户信息业务逻辑
package service

import (
	"context"
	"log/slog"
)

// ===== 请求结构 =====

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	UserID     string
	Username   string
	AvatarData []byte // 头像二进制数据（可选）
	Bio        string
	Grade      string // 年级
}

// ===== 响应结构 =====

// UserProfileResponse 用户信息响应
type UserProfileResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
	Bio       string `json:"bio"`
	Grade     string `json:"grade"`
}

// ===== Service 接口 =====

// UserService 用户信息业务接口
type UserService interface {
	// GetProfile 获取用户信息
	// 输入: userID
	// 输出: 用户详细信息（含头像、简介、年级等）
	GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error)

	// UpdateProfile 更新用户信息
	// 输入: 用户 ID + 待更新字段
	// 操作: 更新 users 和 user_profiles 表
	UpdateProfile(ctx context.Context, req *UpdateProfileRequest) error
}

// ===== 空实现 =====

// userServiceImpl User Service 实现（暂为空实现）
type userServiceImpl struct {
	// TODO: Step2 注入依赖
	// userRepo    db.UserRepository
	// profileRepo db.UserProfileRepository
	// ossProvider domain.OSSProvider（用于上传头像）
	logger *slog.Logger
}

// NewUserService 创建 UserService
func NewUserService(logger *slog.Logger) UserService {
	return &userServiceImpl{logger: logger}
}

func (s *userServiceImpl) GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	// TODO: Step2 实现
	// 1. 查询 users 表获取基本信息
	// 2. 查询 user_profiles 表获取扩展信息
	// 3. 组装返回 UserProfileResponse
	return nil, nil
}

func (s *userServiceImpl) UpdateProfile(ctx context.Context, req *UpdateProfileRequest) error {
	// TODO: Step2 实现
	// 1. 如果有 AvatarData，上传到 OSS 获取 URL
	// 2. 更新 users 表（username, avatar_url）
	// 3. 更新 user_profiles 表（bio, grade）
	return nil
}
