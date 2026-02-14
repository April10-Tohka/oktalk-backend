// Package db 提供对话消息数据库操作
package db

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// ConversationMessageRepository 对话消息数据库操作接口
type ConversationMessageRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, message *model.ConversationMessage) error
	GetByID(ctx context.Context, id string) (*model.ConversationMessage, error)
	Update(ctx context.Context, message *model.ConversationMessage) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByConversationID(ctx context.Context, conversationID string) ([]*model.ConversationMessage, error)
	GetByConversationIDPaginated(ctx context.Context, conversationID string, page, pageSize int) ([]*model.ConversationMessage, int64, error)
	GetLastMessage(ctx context.Context, conversationID string) (*model.ConversationMessage, error)
	GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error)

	// 统计方法
	CountByConversationID(ctx context.Context, conversationID string) (int64, error)
	CountBySenderType(ctx context.Context, conversationID, senderType string) (int64, error)

	// 批量操作
	BatchCreate(ctx context.Context, messages []*model.ConversationMessage) error
	DeleteByConversationID(ctx context.Context, conversationID string) error

	// 事务支持
	WithTx(tx *gorm.DB) ConversationMessageRepository
}

// conversationMessageRepository 对话消息数据库操作实现
type conversationMessageRepository struct {
	db *gorm.DB
}

// NewConversationMessageRepository 创建对话消息数据库操作实例
func NewConversationMessageRepository(db *gorm.DB) ConversationMessageRepository {
	return &conversationMessageRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *conversationMessageRepository) WithTx(tx *gorm.DB) ConversationMessageRepository {
	return &conversationMessageRepository{db: tx}
}

// Create 创建对话消息
func (r *conversationMessageRepository) Create(ctx context.Context, message *model.ConversationMessage) error {
	err := r.db.WithContext(ctx).Create(message).Error
	return WrapDBError(err, "create conversation message")
}

// GetByID 根据 ID 获取对话消息
func (r *conversationMessageRepository) GetByID(ctx context.Context, id string) (*model.ConversationMessage, error) {
	var message model.ConversationMessage
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&message).Error
	if err != nil {
		return nil, WrapDBError(err, "get conversation message by id")
	}
	return &message, nil
}

// Update 更新对话消息
func (r *conversationMessageRepository) Update(ctx context.Context, message *model.ConversationMessage) error {
	err := r.db.WithContext(ctx).Save(message).Error
	return WrapDBError(err, "update conversation message")
}

// Delete 删除对话消息
func (r *conversationMessageRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.ConversationMessage{}).Error
	return WrapDBError(err, "delete conversation message")
}

// GetByConversationID 获取对话的所有消息
func (r *conversationMessageRepository) GetByConversationID(ctx context.Context, conversationID string) ([]*model.ConversationMessage, error) {
	var messages []*model.ConversationMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence_number ASC").
		Find(&messages).Error
	if err != nil {
		return nil, WrapDBError(err, "get conversation messages by conversation id")
	}
	return messages, nil
}

// GetByConversationIDPaginated 分页获取对话的消息
func (r *conversationMessageRepository) GetByConversationIDPaginated(ctx context.Context, conversationID string, page, pageSize int) ([]*model.ConversationMessage, int64, error) {
	var messages []*model.ConversationMessage
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 查询总数
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMessage{}).
		Where("conversation_id = ?", conversationID).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count conversation messages")
	}

	// 查询列表
	err = r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence_number ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list conversation messages")
	}

	return messages, total, nil
}

// GetLastMessage 获取对话的最后一条消息
func (r *conversationMessageRepository) GetLastMessage(ctx context.Context, conversationID string) (*model.ConversationMessage, error) {
	var message model.ConversationMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence_number DESC").
		First(&message).Error
	if err != nil {
		return nil, WrapDBError(err, "get last conversation message")
	}
	return &message, nil
}

// GetNextSequenceNumber 获取下一个消息序号
func (r *conversationMessageRepository) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	var maxSeq int
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMessage{}).
		Where("conversation_id = ?", conversationID).
		Select("COALESCE(MAX(sequence_number), 0)").
		Scan(&maxSeq).Error
	if err != nil {
		return 0, WrapDBError(err, "get next sequence number")
	}
	return maxSeq + 1, nil
}

// CountByConversationID 统计对话消息数
func (r *conversationMessageRepository) CountByConversationID(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMessage{}).
		Where("conversation_id = ?", conversationID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count conversation messages")
	}
	return count, nil
}

// CountBySenderType 统计指定发送者类型的消息数
func (r *conversationMessageRepository) CountBySenderType(ctx context.Context, conversationID, senderType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMessage{}).
		Where("conversation_id = ? AND sender_type = ?", conversationID, senderType).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count conversation messages by sender type")
	}
	return count, nil
}

// BatchCreate 批量创建对话消息
func (r *conversationMessageRepository) BatchCreate(ctx context.Context, messages []*model.ConversationMessage) error {
	if len(messages) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(messages, 100).Error
	return WrapDBError(err, "batch create conversation messages")
}

// DeleteByConversationID 删除对话的所有消息
func (r *conversationMessageRepository) DeleteByConversationID(ctx context.Context, conversationID string) error {
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Delete(&model.ConversationMessage{}).Error
	return WrapDBError(err, "delete conversation messages by conversation id")
}
