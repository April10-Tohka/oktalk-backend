// Package db 提供用户登录日志数据库操作
package db

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// UserLoginLogRepository 用户登录日志数据库操作接口
type UserLoginLogRepository interface {
	// Insert 创建登录日志记录
	Insert(ctx context.Context, logEntry *model.UserLoginLog) error
	// ListByUserID 根据用户 ID 查询登录日志列表
	ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.UserLoginLog, int64, error)
	// WithTx 返回使用事务的 Repository
	WithTx(tx *gorm.DB) UserLoginLogRepository
}

// userLoginLogRepository 用户登录日志数据库操作实现
type userLoginLogRepository struct {
	db *gorm.DB
}

// NewUserLoginLogRepository 创建用户登录日志数据库操作实例
func NewUserLoginLogRepository(db *gorm.DB) UserLoginLogRepository {
	return &userLoginLogRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *userLoginLogRepository) WithTx(tx *gorm.DB) UserLoginLogRepository {
	return &userLoginLogRepository{db: tx}
}

// Insert 创建登录日志记录
func (r *userLoginLogRepository) Insert(ctx context.Context, logEntry *model.UserLoginLog) error {
	err := r.db.WithContext(ctx).Create(logEntry).Error
	return WrapDBError(err, "insert login log")
}

// ListByUserID 根据用户 ID 查询登录日志列表
func (r *userLoginLogRepository) ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.UserLoginLog, int64, error) {
	var logs []*model.UserLoginLog
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.UserLoginLog{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count login logs")
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list login logs")
	}

	return logs, total, nil
}
