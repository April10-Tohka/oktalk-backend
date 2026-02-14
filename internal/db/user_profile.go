// Package db 提供用户扩展信息数据库操作
package db

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// UserProfileRepository 用户扩展信息数据库操作接口
type UserProfileRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, profile *model.UserProfile) error
	GetByID(ctx context.Context, id string) (*model.UserProfile, error)
	GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error)
	Update(ctx context.Context, profile *model.UserProfile) error
	Delete(ctx context.Context, id string) error

	// 统计更新方法
	IncrementConversations(ctx context.Context, userID string) error
	IncrementEvaluations(ctx context.Context, userID string) error
	IncrementReports(ctx context.Context, userID string) error
	UpdateAverageScore(ctx context.Context, userID string, score float64) error
	UpdateLastConversationAt(ctx context.Context, userID string) error
	UpdateLastEvaluationAt(ctx context.Context, userID string) error

	// 事务支持
	WithTx(tx *gorm.DB) UserProfileRepository
}

// userProfileRepository 用户扩展信息数据库操作实现
type userProfileRepository struct {
	db *gorm.DB
}

// NewUserProfileRepository 创建用户扩展信息数据库操作实例
func NewUserProfileRepository(db *gorm.DB) UserProfileRepository {
	return &userProfileRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *userProfileRepository) WithTx(tx *gorm.DB) UserProfileRepository {
	return &userProfileRepository{db: tx}
}

// Create 创建用户扩展信息
func (r *userProfileRepository) Create(ctx context.Context, profile *model.UserProfile) error {
	err := r.db.WithContext(ctx).Create(profile).Error
	return WrapDBError(err, "create user profile")
}

// GetByID 根据 ID 获取用户扩展信息
func (r *userProfileRepository) GetByID(ctx context.Context, id string) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&profile).Error
	if err != nil {
		return nil, WrapDBError(err, "get user profile by id")
	}
	return &profile, nil
}

// GetByUserID 根据用户 ID 获取扩展信息
func (r *userProfileRepository) GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&profile).Error
	if err != nil {
		return nil, WrapDBError(err, "get user profile by user id")
	}
	return &profile, nil
}

// Update 更新用户扩展信息
func (r *userProfileRepository) Update(ctx context.Context, profile *model.UserProfile) error {
	err := r.db.WithContext(ctx).Save(profile).Error
	return WrapDBError(err, "update user profile")
}

// Delete 删除用户扩展信息
func (r *userProfileRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.UserProfile{}).Error
	return WrapDBError(err, "delete user profile")
}

// IncrementConversations 增加用户对话数
func (r *userProfileRepository) IncrementConversations(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_conversations", gorm.Expr("total_conversations + ?", 1)).Error
	return WrapDBError(err, "increment user conversations")
}

// IncrementEvaluations 增加用户评测数
func (r *userProfileRepository) IncrementEvaluations(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_evaluations", gorm.Expr("total_evaluations + ?", 1)).Error
	return WrapDBError(err, "increment user evaluations")
}

// IncrementReports 增加用户报告数
func (r *userProfileRepository) IncrementReports(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_reports", gorm.Expr("total_reports + ?", 1)).Error
	return WrapDBError(err, "increment user reports")
}

// UpdateAverageScore 更新用户平均评测分数
func (r *userProfileRepository) UpdateAverageScore(ctx context.Context, userID string, score float64) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		Update("average_evaluation_score", score).Error
	return WrapDBError(err, "update user average score")
}

// UpdateLastConversationAt 更新用户最后对话时间
func (r *userProfileRepository) UpdateLastConversationAt(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		Update("last_conversation_at", gorm.Expr("NOW()")).Error
	return WrapDBError(err, "update user last conversation at")
}

// UpdateLastEvaluationAt 更新用户最后评测时间
func (r *userProfileRepository) UpdateLastEvaluationAt(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		Update("last_evaluation_at", gorm.Expr("NOW()")).Error
	return WrapDBError(err, "update user last evaluation at")
}
