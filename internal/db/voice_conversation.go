// Package db 提供语音对话数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// VoiceConversationRepository 语音对话数据库操作接口
type VoiceConversationRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, conversation *model.VoiceConversation) error
	GetByID(ctx context.Context, id string) (*model.VoiceConversation, error)
	Update(ctx context.Context, conversation *model.VoiceConversation) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.VoiceConversation, int64, error)
	GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.VoiceConversation, int64, error)
	GetByUserIDAndStatus(ctx context.Context, userID, status string) ([]*model.VoiceConversation, error)

	// 统计方法
	Count(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CountByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (int64, error)

	// 更新方法
	UpdateStatus(ctx context.Context, id, status string) error
	IncrementMessageCount(ctx context.Context, id string) error
	UpdateDuration(ctx context.Context, id string, duration int) error
	UpdateScore(ctx context.Context, id string, score int) error

	// 预加载方法
	GetWithMessages(ctx context.Context, id string) (*model.VoiceConversation, error)

	// 事务支持
	WithTx(tx *gorm.DB) VoiceConversationRepository
}

// voiceConversationRepository 语音对话数据库操作实现
type voiceConversationRepository struct {
	db *gorm.DB
}

// NewVoiceConversationRepository 创建语音对话数据库操作实例
func NewVoiceConversationRepository(db *gorm.DB) VoiceConversationRepository {
	return &voiceConversationRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *voiceConversationRepository) WithTx(tx *gorm.DB) VoiceConversationRepository {
	return &voiceConversationRepository{db: tx}
}

// Create 创建语音对话
func (r *voiceConversationRepository) Create(ctx context.Context, conversation *model.VoiceConversation) error {
	err := r.db.WithContext(ctx).Create(conversation).Error
	return WrapDBError(err, "create voice conversation")
}

// GetByID 根据 ID 获取语音对话
func (r *voiceConversationRepository) GetByID(ctx context.Context, id string) (*model.VoiceConversation, error) {
	var conversation model.VoiceConversation
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&conversation).Error
	if err != nil {
		return nil, WrapDBError(err, "get voice conversation by id")
	}
	return &conversation, nil
}

// Update 更新语音对话
func (r *voiceConversationRepository) Update(ctx context.Context, conversation *model.VoiceConversation) error {
	err := r.db.WithContext(ctx).Save(conversation).Error
	return WrapDBError(err, "update voice conversation")
}

// Delete 软删除语音对话
func (r *voiceConversationRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now).Error
	return WrapDBError(err, "delete voice conversation")
}

// GetByUserID 根据用户 ID 分页获取对话列表
func (r *voiceConversationRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.VoiceConversation, int64, error) {
	var conversations []*model.VoiceConversation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 查询总数
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count voice conversations by user id")
	}

	// 查询列表
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&conversations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list voice conversations by user id")
	}

	return conversations, total, nil
}

// GetByStatus 根据状态分页获取对话列表
func (r *voiceConversationRepository) GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.VoiceConversation, int64, error) {
	var conversations []*model.VoiceConversation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("status = ? AND deleted_at IS NULL", status).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count voice conversations by status")
	}

	err = r.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", status).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&conversations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list voice conversations by status")
	}

	return conversations, total, nil
}

// GetByUserIDAndStatus 根据用户 ID 和状态获取对话列表
func (r *voiceConversationRepository) GetByUserIDAndStatus(ctx context.Context, userID, status string) ([]*model.VoiceConversation, error) {
	var conversations []*model.VoiceConversation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND deleted_at IS NULL", userID, status).
		Order("created_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, WrapDBError(err, "get voice conversations by user id and status")
	}
	return conversations, nil
}

// Count 统计对话总数
func (r *voiceConversationRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count voice conversations")
	}
	return count, nil
}

// CountByUserID 统计用户对话数
func (r *voiceConversationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count voice conversations by user id")
	}
	return count, nil
}

// CountByUserIDAndDateRange 统计用户在指定时间范围内的对话数
func (r *voiceConversationRepository) CountByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ? AND deleted_at IS NULL", userID, start, end).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count voice conversations by user id and date range")
	}
	return count, nil
}

// UpdateStatus 更新对话状态
func (r *voiceConversationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("status", status).Error
	return WrapDBError(err, "update voice conversation status")
}

// IncrementMessageCount 增加消息数
func (r *voiceConversationRepository) IncrementMessageCount(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("id = ? AND deleted_at IS NULL", id).
		UpdateColumn("message_count", gorm.Expr("message_count + ?", 1)).Error
	return WrapDBError(err, "increment voice conversation message count")
}

// UpdateDuration 更新对话时长
func (r *voiceConversationRepository) UpdateDuration(ctx context.Context, id string, duration int) error {
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("duration_seconds", duration).Error
	return WrapDBError(err, "update voice conversation duration")
}

// UpdateScore 更新对话评分
func (r *voiceConversationRepository) UpdateScore(ctx context.Context, id string, score int) error {
	err := r.db.WithContext(ctx).
		Model(&model.VoiceConversation{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("score", score).Error
	return WrapDBError(err, "update voice conversation score")
}

// GetWithMessages 获取对话及其消息
func (r *voiceConversationRepository) GetWithMessages(ctx context.Context, id string) (*model.VoiceConversation, error) {
	var conversation model.VoiceConversation
	err := r.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence_number ASC")
		}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&conversation).Error
	if err != nil {
		return nil, WrapDBError(err, "get voice conversation with messages")
	}
	return &conversation, nil
}
