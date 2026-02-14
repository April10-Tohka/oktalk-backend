// Package db 提供系统配置数据库操作
package db

import (
	"context"
	"strconv"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// SystemSettingRepository 系统配置数据库操作接口
type SystemSettingRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, setting *model.SystemSetting) error
	GetByID(ctx context.Context, id string) (*model.SystemSetting, error)
	GetByKey(ctx context.Context, key string) (*model.SystemSetting, error)
	Update(ctx context.Context, setting *model.SystemSetting) error
	Delete(ctx context.Context, id string) error
	DeleteByKey(ctx context.Context, key string) error

	// 查询方法
	GetAll(ctx context.Context) ([]*model.SystemSetting, error)
	GetByType(ctx context.Context, configType string) ([]*model.SystemSetting, error)
	GetEditable(ctx context.Context) ([]*model.SystemSetting, error)

	// 值操作方法
	GetValue(ctx context.Context, key string) (string, error)
	GetIntValue(ctx context.Context, key string) (int, error)
	GetFloatValue(ctx context.Context, key string) (float64, error)
	GetBoolValue(ctx context.Context, key string) (bool, error)
	SetValue(ctx context.Context, key, value string) error

	// 批量操作
	BatchCreate(ctx context.Context, settings []*model.SystemSetting) error
	BatchUpdate(ctx context.Context, settings []*model.SystemSetting) error

	// 初始化
	InitDefaults(ctx context.Context) error

	// 事务支持
	WithTx(tx *gorm.DB) SystemSettingRepository
}

// systemSettingRepository 系统配置数据库操作实现
type systemSettingRepository struct {
	db *gorm.DB
}

// NewSystemSettingRepository 创建系统配置数据库操作实例
func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *systemSettingRepository) WithTx(tx *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: tx}
}

// Create 创建系统配置
func (r *systemSettingRepository) Create(ctx context.Context, setting *model.SystemSetting) error {
	err := r.db.WithContext(ctx).Create(setting).Error
	return WrapDBError(err, "create system setting")
}

// GetByID 根据 ID 获取系统配置
func (r *systemSettingRepository) GetByID(ctx context.Context, id string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&setting).Error
	if err != nil {
		return nil, WrapDBError(err, "get system setting by id")
	}
	return &setting, nil
}

// GetByKey 根据配置键获取系统配置
func (r *systemSettingRepository) GetByKey(ctx context.Context, key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("config_key = ?", key).
		First(&setting).Error
	if err != nil {
		return nil, WrapDBError(err, "get system setting by key")
	}
	return &setting, nil
}

// Update 更新系统配置
func (r *systemSettingRepository) Update(ctx context.Context, setting *model.SystemSetting) error {
	err := r.db.WithContext(ctx).Save(setting).Error
	return WrapDBError(err, "update system setting")
}

// Delete 根据 ID 删除系统配置
func (r *systemSettingRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.SystemSetting{}).Error
	return WrapDBError(err, "delete system setting")
}

// DeleteByKey 根据配置键删除系统配置
func (r *systemSettingRepository) DeleteByKey(ctx context.Context, key string) error {
	err := r.db.WithContext(ctx).
		Where("config_key = ?", key).
		Delete(&model.SystemSetting{}).Error
	return WrapDBError(err, "delete system setting by key")
}

// GetAll 获取所有系统配置
func (r *systemSettingRepository) GetAll(ctx context.Context) ([]*model.SystemSetting, error) {
	var settings []*model.SystemSetting
	err := r.db.WithContext(ctx).
		Order("config_key ASC").
		Find(&settings).Error
	if err != nil {
		return nil, WrapDBError(err, "get all system settings")
	}
	return settings, nil
}

// GetByType 根据配置类型获取系统配置
func (r *systemSettingRepository) GetByType(ctx context.Context, configType string) ([]*model.SystemSetting, error) {
	var settings []*model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("config_type = ?", configType).
		Order("config_key ASC").
		Find(&settings).Error
	if err != nil {
		return nil, WrapDBError(err, "get system settings by type")
	}
	return settings, nil
}

// GetEditable 获取可编辑的系统配置
func (r *systemSettingRepository) GetEditable(ctx context.Context) ([]*model.SystemSetting, error) {
	var settings []*model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("is_editable = ?", true).
		Order("config_key ASC").
		Find(&settings).Error
	if err != nil {
		return nil, WrapDBError(err, "get editable system settings")
	}
	return settings, nil
}

// GetValue 获取配置值（字符串）
func (r *systemSettingRepository) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.ConfigValue, nil
}

// GetIntValue 获取配置值（整数）
func (r *systemSettingRepository) GetIntValue(ctx context.Context, key string) (int, error) {
	value, err := r.GetValue(ctx, key)
	if err != nil {
		return 0, err
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, WrapDBError(err, "parse int value")
	}
	return intValue, nil
}

// GetFloatValue 获取配置值（浮点数）
func (r *systemSettingRepository) GetFloatValue(ctx context.Context, key string) (float64, error) {
	value, err := r.GetValue(ctx, key)
	if err != nil {
		return 0, err
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, WrapDBError(err, "parse float value")
	}
	return floatValue, nil
}

// GetBoolValue 获取配置值（布尔值）
func (r *systemSettingRepository) GetBoolValue(ctx context.Context, key string) (bool, error) {
	value, err := r.GetValue(ctx, key)
	if err != nil {
		return false, err
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, WrapDBError(err, "parse bool value")
	}
	return boolValue, nil
}

// SetValue 设置配置值
func (r *systemSettingRepository) SetValue(ctx context.Context, key, value string) error {
	err := r.db.WithContext(ctx).
		Model(&model.SystemSetting{}).
		Where("config_key = ?", key).
		Update("config_value", value).Error
	return WrapDBError(err, "set system setting value")
}

// BatchCreate 批量创建系统配置
func (r *systemSettingRepository) BatchCreate(ctx context.Context, settings []*model.SystemSetting) error {
	if len(settings) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(settings, 100).Error
	return WrapDBError(err, "batch create system settings")
}

// BatchUpdate 批量更新系统配置
func (r *systemSettingRepository) BatchUpdate(ctx context.Context, settings []*model.SystemSetting) error {
	if len(settings) == 0 {
		return nil
	}
	// 使用事务批量更新
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, setting := range settings {
			if err := tx.Save(setting).Error; err != nil {
				return WrapDBError(err, "batch update system settings")
			}
		}
		return nil
	})
}

// InitDefaults 初始化默认系统配置
// 只创建不存在的配置，不覆盖已存在的配置
func (r *systemSettingRepository) InitDefaults(ctx context.Context) error {
	for _, defaultSetting := range model.DefaultSystemSettings {
		// 检查是否已存在
		var count int64
		err := r.db.WithContext(ctx).
			Model(&model.SystemSetting{}).
			Where("config_key = ?", defaultSetting.ConfigKey).
			Count(&count).Error
		if err != nil {
			return WrapDBError(err, "check system setting exists")
		}

		// 不存在则创建
		if count == 0 {
			setting := defaultSetting // 复制以避免修改原始数据
			if err := r.db.WithContext(ctx).Create(&setting).Error; err != nil {
				return WrapDBError(err, "init default system setting")
			}
		}
	}
	return nil
}
