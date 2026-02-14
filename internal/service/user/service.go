// Package user 提供用户业务逻辑
package user

import (
	"context"

	"pronunciation-correction-system/internal/model"
)

// ServiceImpl 用户服务实现
type ServiceImpl struct {
	// TODO: 添加依赖
	// userDB    db.UserRepository
	// userCache cache.UserCache
}

// NewService 创建用户服务
func NewService() Service {
	return &ServiceImpl{}
}

// Register 用户注册
func (s *ServiceImpl) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	// TODO: 实现用户注册
	// 1. 验证输入参数
	// 2. 检查用户是否已存在
	// 3. 密码加密
	// 4. 创建用户记录
	// 5. 返回用户信息
	return nil, nil
}

// Login 用户登录
func (s *ServiceImpl) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// TODO: 实现用户登录
	// 1. 验证用户凭证
	// 2. 生成 JWT Token
	// 3. 更新登录时间
	// 4. 返回 Token
	return nil, nil
}

// GetProfile 获取用户信息
func (s *ServiceImpl) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	// TODO: 实现获取用户信息
	return nil, nil
}

// UpdateProfile 更新用户信息
func (s *ServiceImpl) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*model.User, error) {
	// TODO: 实现更新用户信息
	return nil, nil
}

// GetLearningStats 获取学习统计
func (s *ServiceImpl) GetLearningStats(ctx context.Context, userID string) (*LearningStats, error) {
	// TODO: 实现获取学习统计
	return nil, nil
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      *model.User `json:"user"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Level     string `json:"level"`
}

// LearningStats 学习统计
type LearningStats struct {
	TotalEvaluations int     `json:"total_evaluations"`
	AverageScore     float64 `json:"average_score"`
	TotalDuration    int     `json:"total_duration"` // 总练习时长（秒）
	ConsecutiveDays  int     `json:"consecutive_days"`
	WeeklyProgress   []int   `json:"weekly_progress"`
}
