// Package service 提供用户信息业务逻辑
package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/db"
)

// ===================== 请求结构 =====================

// UpdateProfileRequest 更新用户信息请求（所有字段为可选，使用指针类型）
type UpdateProfileRequest struct {
	UserID    string  `json:"-"` // 由 middleware 注入
	Username  *string `json:"username,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Grade     *int    `json:"grade,omitempty"`
	Age       *int    `json:"age,omitempty"`
	Gender    *string `json:"gender,omitempty"`
	Bio       *string `json:"bio,omitempty"`
}

// ===================== 响应结构 =====================

// UserProfileResponse 用户信息响应
type UserProfileResponse struct {
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	Phone     string  `json:"phone,omitempty"` // 脱敏后
	AvatarURL *string `json:"avatar_url,omitempty"`
	Status    string  `json:"status"`
	Grade     *int    `json:"grade,omitempty"`
	CreatedAt string  `json:"created_at"`
	Profile   *UserProfileDetail `json:"profile,omitempty"`
}

// UserProfileDetail 用户扩展信息
type UserProfileDetail struct {
	Age                    *int     `json:"age,omitempty"`
	Gender                 *string  `json:"gender,omitempty"`
	Bio                    *string  `json:"bio,omitempty"`
	TotalConversations     int      `json:"total_conversations"`
	TotalEvaluations       int      `json:"total_evaluations"`
	TotalReports           int      `json:"total_reports"`
	TotalStudyMinutes      int      `json:"total_study_minutes"`
	AverageEvaluationScore float64  `json:"average_evaluation_score"`
	LastConversationAt     *string  `json:"last_conversation_at,omitempty"`
	LastEvaluationAt       *string  `json:"last_evaluation_at,omitempty"`
}

// ===================== Service 接口 =====================

// UserService 用户信息业务接口
type UserService interface {
	// GetProfile 获取用户信息
	GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error)
	// UpdateProfile 更新用户信息
	UpdateProfile(ctx context.Context, req *UpdateProfileRequest) (*UserProfileResponse, error)
}

// ===================== 实现 =====================

type userServiceImpl struct {
	database *gorm.DB
	rdb      *redis.Client
	repos    *db.Repositories
	logger   *slog.Logger
}

// NewUserService 创建 UserService
func NewUserService(
	database *gorm.DB,
	rdb *redis.Client,
	repos *db.Repositories,
	logger *slog.Logger,
) UserService {
	return &userServiceImpl{
		database: database,
		rdb:      rdb,
		repos:    repos,
		logger:   logger,
	}
}

// ===================== 6.6 GetProfile 获取个人资料 =====================

func (s *userServiceImpl) GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	if userID == "" {
		return nil, &AuthError{Code: 400, Message: "user_id 不能为空"}
	}

	// 1. 查缓存
	profileKey := fmt.Sprintf(cache.KeyUserProfile, userID)
	if s.rdb != nil {
		cached, found, err := cache.GetJSON[UserProfileResponse](ctx, s.rdb, profileKey)
		if err != nil {
			s.logger.Warn("get profile cache failed", slog.String("error", err.Error()))
		}
		if found && cached != nil {
			return cached, nil
		}
	}

	// 2. 缓存未命中，查数据库
	resp, err := s.queryProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3. 写缓存
	if s.rdb != nil {
		if err := cache.SetJSON(ctx, s.rdb, profileKey, resp, cache.TTLUserProfile); err != nil {
			s.logger.Warn("set profile cache failed", slog.String("error", err.Error()))
		}
	}

	return resp, nil
}

// queryProfile 从数据库查询用户完整资料
func (s *userServiceImpl) queryProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	user, err := s.repos.User.GetWithProfile(ctx, userID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, &AuthError{Code: 404, Message: "用户不存在"}
		}
		return nil, &AuthError{Code: 500, Message: "查询用户失败"}
	}

	resp := &UserProfileResponse{
		UserID:    user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Status:    user.Status,
		Grade:     user.Grade,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// 手机号脱敏
	if user.Phone != nil && *user.Phone != "" {
		resp.Phone = maskPhone(*user.Phone)
	}

	// 扩展信息
	if user.Profile != nil {
		p := user.Profile
		detail := &UserProfileDetail{
			Age:                    p.Age,
			Gender:                 p.Gender,
			Bio:                    p.Bio,
			TotalConversations:     p.TotalConversations,
			TotalEvaluations:       p.TotalEvaluations,
			TotalReports:           p.TotalReports,
			TotalStudyMinutes:      p.TotalStudyMinutes,
			AverageEvaluationScore: p.AverageEvaluationScore,
		}
		if p.LastConversationAt != nil {
			t := p.LastConversationAt.Format("2006-01-02T15:04:05Z")
			detail.LastConversationAt = &t
		}
		if p.LastEvaluationAt != nil {
			t := p.LastEvaluationAt.Format("2006-01-02T15:04:05Z")
			detail.LastEvaluationAt = &t
		}
		resp.Profile = detail
	}

	return resp, nil
}

// ===================== 6.7 UpdateProfile 更新个人资料 =====================

// 特殊字符正则：不允许 < > ' " &
var specialCharRegexp = regexp.MustCompile(`[<>'"&]`)

func (s *userServiceImpl) UpdateProfile(ctx context.Context, req *UpdateProfileRequest) (*UserProfileResponse, error) {
	if req.UserID == "" {
		return nil, &AuthError{Code: 400, Message: "user_id 不能为空"}
	}

	// 1. 字段校验（只校验传入的字段）
	if req.Username != nil {
		u := strings.TrimSpace(*req.Username)
		if len(u) < 2 || len(u) > 50 {
			return nil, &AuthError{Code: 400, Message: "用户名长度需在 2-50 字符之间"}
		}
		if specialCharRegexp.MatchString(u) {
			return nil, &AuthError{Code: 400, Message: "用户名不能包含特殊字符"}
		}
	}
	if req.AvatarURL != nil {
		if len(*req.AvatarURL) > 500 {
			return nil, &AuthError{Code: 400, Message: "头像 URL 长度不能超过 500 字符"}
		}
	}
	if req.Grade != nil {
		if *req.Grade < 1 || *req.Grade > 6 {
			return nil, &AuthError{Code: 400, Message: "年级范围为 1-6"}
		}
	}
	if req.Age != nil {
		if *req.Age < 1 || *req.Age > 18 {
			return nil, &AuthError{Code: 400, Message: "年龄范围为 1-18"}
		}
	}
	if req.Gender != nil {
		if *req.Gender != "male" && *req.Gender != "female" {
			return nil, &AuthError{Code: 400, Message: "性别只能是 male 或 female"}
		}
	}
	if req.Bio != nil {
		if len(*req.Bio) > 500 {
			return nil, &AuthError{Code: 400, Message: "个人简介长度不能超过 500 字符"}
		}
	}

	// 2. 若传入 username，检查唯一性
	if req.Username != nil {
		existing, err := s.repos.User.GetByUsername(ctx, *req.Username)
		if err != nil && !db.IsNotFound(err) {
			return nil, &AuthError{Code: 500, Message: "检查用户名失败"}
		}
		if existing != nil && existing.ID != req.UserID {
			return nil, &AuthError{Code: 409, Message: "用户名已被使用"}
		}
	}

	// 3. 构建更新 Map
	usersUpdate := map[string]interface{}{}
	profileUpdate := map[string]interface{}{}

	if req.Username != nil {
		usersUpdate["username"] = strings.TrimSpace(*req.Username)
	}
	if req.AvatarURL != nil {
		usersUpdate["avatar_url"] = *req.AvatarURL
	}
	if req.Grade != nil {
		usersUpdate["grade"] = *req.Grade
	}
	if req.Age != nil {
		profileUpdate["age"] = *req.Age
	}
	if req.Gender != nil {
		profileUpdate["gender"] = *req.Gender
	}
	if req.Bio != nil {
		profileUpdate["bio"] = *req.Bio
	}

	// 4. 事务中更新
	if len(usersUpdate) > 0 || len(profileUpdate) > 0 {
		err := s.database.Transaction(func(tx *gorm.DB) error {
			if len(usersUpdate) > 0 {
				if err := tx.Model(&struct{ ID string }{}).
					Table("users").
					Where("id = ? AND deleted_at IS NULL", req.UserID).
					Updates(usersUpdate).Error; err != nil {
					return err
				}
			}
			if len(profileUpdate) > 0 {
				if err := tx.Table("user_profiles").
					Where("user_id = ?", req.UserID).
					Updates(profileUpdate).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			s.logger.Error("update profile failed", slog.String("error", err.Error()))
			return nil, &AuthError{Code: 500, Message: "更新资料失败"}
		}
	}

	// 5. 清除缓存
	if s.rdb != nil {
		profileKey := fmt.Sprintf(cache.KeyUserProfile, req.UserID)
		s.rdb.Del(ctx, profileKey)
	}

	// 6. 复用 GetProfile 查询并返回最新资料
	return s.queryProfile(ctx, req.UserID)
}
