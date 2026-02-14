// Package user 定义用户服务接口
package user

import (
	"context"

	"pronunciation-correction-system/internal/model"
)

// Service 用户服务接口
type Service interface {
	// Register 用户注册
	Register(ctx context.Context, req *RegisterRequest) (*model.User, error)

	// Login 用户登录
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)

	// GetProfile 获取用户信息
	GetProfile(ctx context.Context, userID string) (*model.User, error)

	// UpdateProfile 更新用户信息
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*model.User, error)

	// GetLearningStats 获取学习统计
	GetLearningStats(ctx context.Context, userID string) (*LearningStats, error)
}
