// Package db 提供用户数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// UserRepository 用户数据库操作接口
type UserRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByPhone(ctx context.Context, phone string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.User, int64, error)

	// 统计方法
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)

	// 预加载方法
	GetWithProfile(ctx context.Context, id string) (*model.User, error)

	// 事务支持
	WithTx(tx *gorm.DB) UserRepository

	// 软删除恢复
	Restore(ctx context.Context, id string) error

	// 批量操作
	BatchCreate(ctx context.Context, users []*model.User) error
	BatchUpdateStatus(ctx context.Context, ids []string, status string) error
}

// userRepository 用户数据库操作实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据库操作实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *userRepository) WithTx(tx *gorm.DB) UserRepository {
	return &userRepository{db: tx}
}

// Create 创建用户
// 自动设置创建时间和更新时间
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	return WrapDBError(err, "create user")
}

// GetByID 根据 ID 获取用户
// 自动排除软删除的记录
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		return nil, WrapDBError(err, "get user by id")
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		return nil, WrapDBError(err, "get user by email")
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ? AND deleted_at IS NULL", username).
		First(&user).Error
	if err != nil {
		return nil, WrapDBError(err, "get user by username")
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("phone = ? AND deleted_at IS NULL", phone).
		First(&user).Error
	if err != nil {
		return nil, WrapDBError(err, "get user by phone")
	}
	return &user, nil
}

// Update 更新用户
// 使用 Save 方法会更新所有字段，包括零值
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Save(user).Error
	return WrapDBError(err, "update user")
}

// Delete 软删除用户
// 通过设置 deleted_at 实现软删除
func (r *userRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now).Error
	return WrapDBError(err, "delete user")
}

// Restore 恢复软删除的用户
func (r *userRepository) Restore(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
	return WrapDBError(err, "restore user")
}

// List 分页获取用户列表
// 返回用户列表、总数和错误
func (r *userRepository) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 查询总数
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("deleted_at IS NULL").
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count users")
	}

	// 查询列表
	err = r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list users")
	}

	return users, total, nil
}

// ListByStatus 根据状态分页获取用户列表
func (r *userRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 查询总数
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("status = ? AND deleted_at IS NULL", status).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count users by status")
	}

	// 查询列表
	err = r.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", status).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list users by status")
	}

	return users, total, nil
}

// Count 统计用户总数
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count users")
	}
	return count, nil
}

// CountByStatus 根据状态统计用户数量
func (r *userRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("status = ? AND deleted_at IS NULL", status).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count users by status")
	}
	return count, nil
}

// GetWithProfile 获取用户及其扩展信息
func (r *userRepository) GetWithProfile(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		return nil, WrapDBError(err, "get user with profile")
	}
	return &user, nil
}

// BatchCreate 批量创建用户
func (r *userRepository) BatchCreate(ctx context.Context, users []*model.User) error {
	if len(users) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(users, 100).Error
	return WrapDBError(err, "batch create users")
}

// BatchUpdateStatus 批量更新用户状态
func (r *userRepository) BatchUpdateStatus(ctx context.Context, ids []string, status string) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("status", status).Error
	return WrapDBError(err, "batch update user status")
}
