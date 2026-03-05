// Package db 提供微信绑定数据库操作
package db

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// UserWechatBindingRepository 微信绑定数据库操作接口
type UserWechatBindingRepository interface {
	// Create 创建微信绑定记录
	Create(ctx context.Context, binding *model.UserWechatBinding) error
	// FindByOpenID 根据微信 OpenID 查询绑定关系
	FindByOpenID(ctx context.Context, openID string) (*model.UserWechatBinding, error)
	// FindByUserID 根据用户 ID 查询绑定关系
	FindByUserID(ctx context.Context, userID string) (*model.UserWechatBinding, error)
	// Update 更新绑定信息
	Update(ctx context.Context, binding *model.UserWechatBinding) error
	// WithTx 返回使用事务的 Repository
	WithTx(tx *gorm.DB) UserWechatBindingRepository
}

// userWechatBindingRepository 微信绑定数据库操作实现
type userWechatBindingRepository struct {
	db *gorm.DB
}

// NewUserWechatBindingRepository 创建微信绑定数据库操作实例
func NewUserWechatBindingRepository(db *gorm.DB) UserWechatBindingRepository {
	return &userWechatBindingRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *userWechatBindingRepository) WithTx(tx *gorm.DB) UserWechatBindingRepository {
	return &userWechatBindingRepository{db: tx}
}

// Create 创建微信绑定记录
func (r *userWechatBindingRepository) Create(ctx context.Context, binding *model.UserWechatBinding) error {
	err := r.db.WithContext(ctx).Create(binding).Error
	return WrapDBError(err, "create wechat binding")
}

// FindByOpenID 根据微信 OpenID 查询绑定关系
func (r *userWechatBindingRepository) FindByOpenID(ctx context.Context, openID string) (*model.UserWechatBinding, error) {
	var binding model.UserWechatBinding
	err := r.db.WithContext(ctx).
		Where("open_id = ?", openID).
		First(&binding).Error
	if err != nil {
		return nil, WrapDBError(err, "find wechat binding by open_id")
	}
	return &binding, nil
}

// FindByUserID 根据用户 ID 查询绑定关系
func (r *userWechatBindingRepository) FindByUserID(ctx context.Context, userID string) (*model.UserWechatBinding, error) {
	var binding model.UserWechatBinding
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&binding).Error
	if err != nil {
		return nil, WrapDBError(err, "find wechat binding by user_id")
	}
	return &binding, nil
}

// Update 更新绑定信息
func (r *userWechatBindingRepository) Update(ctx context.Context, binding *model.UserWechatBinding) error {
	err := r.db.WithContext(ctx).Save(binding).Error
	return WrapDBError(err, "update wechat binding")
}
