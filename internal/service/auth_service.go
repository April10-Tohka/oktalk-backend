// Package service 提供认证业务逻辑
package service

import (
	"context"
	"log/slog"
)

// ===== 请求结构 =====

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// ===== 响应结构 =====

// AuthResponse 认证响应（登录 / 注册 通用）
type AuthResponse struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AvatarURL    string `json:"avatar_url"`
}

// TokenResponse Token 刷新响应
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// ===== Service 接口 =====

// AuthService 认证业务接口
type AuthService interface {
	// Login 用户登录
	// 输入: email + password
	// 输出: 用户信息 + JWT Token
	Login(ctx context.Context, email, password string) (*AuthResponse, error)

	// Register 用户注册
	// 输入: email + password + username
	// 输出: 用户信息 + JWT Token（注册后自动登录）
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)

	// Logout 用户登出
	// 输入: userID + token
	// 操作: 将 token 加入黑名单
	Logout(ctx context.Context, userID, token string) error

	// RefreshToken 刷新 Token
	// 输入: refreshToken
	// 输出: 新的 Token + 过期时间
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
}

// ===== 空实现 =====

// authServiceImpl Auth Service 实现（暂为空实现）
type authServiceImpl struct {
	// TODO: Step2 注入依赖
	// userRepo   db.UserRepository
	// jwtSecret  string
	// redisClient *redis.Client（用于 token 黑名单）
	logger *slog.Logger
}

// NewAuthService 创建 AuthService
func NewAuthService(logger *slog.Logger) AuthService {
	return &authServiceImpl{logger: logger}
}

func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	// TODO: Step2 实现
	// 1. 根据 email 查询用户
	// 2. 验证密码（bcrypt.CompareHashAndPassword）
	// 3. 生成 JWT Token + RefreshToken
	// 4. 返回用户信息和 Token
	return nil, nil
}

func (s *authServiceImpl) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// TODO: Step2 实现
	// 1. 检查 email 是否已注册
	// 2. 密码加密（bcrypt）
	// 3. 创建用户记录
	// 4. 创建用户画像（UserProfile）
	// 5. 生成 JWT Token + RefreshToken
	// 6. 返回用户信息和 Token
	return nil, nil
}

func (s *authServiceImpl) Logout(ctx context.Context, userID, token string) error {
	// TODO: Step2 实现
	// 1. 将 token 加入 Redis 黑名单
	// 2. 设置过期时间 = token 剩余有效期
	return nil
}

func (s *authServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// TODO: Step2 实现
	// 1. 验证 refreshToken 合法性
	// 2. 检查 refreshToken 是否在黑名单
	// 3. 生成新的 Token
	// 4. 返回新 Token + 过期时间
	return nil, nil
}
