// Package db 提供反馈记录数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// FeedbackRecordRepository 反馈记录数据库操作接口
type FeedbackRecordRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, record *model.FeedbackRecord) error
	GetByID(ctx context.Context, id string) (*model.FeedbackRecord, error)
	GetByEvaluationID(ctx context.Context, evaluationID string) (*model.FeedbackRecord, error)
	Update(ctx context.Context, record *model.FeedbackRecord) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.FeedbackRecord, int64, error)
	GetByFeedbackLevel(ctx context.Context, level string, page, pageSize int) ([]*model.FeedbackRecord, int64, error)
	GetByGenerationMethod(ctx context.Context, method string, page, pageSize int) ([]*model.FeedbackRecord, int64, error)

	// 统计方法
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountByFeedbackLevel(ctx context.Context, level string) (int64, error)
	GetAverageGenerationDuration(ctx context.Context) (float64, error)
	GetAverageQualityScore(ctx context.Context) (float64, error)

	// 更新方法
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateQualityScore(ctx context.Context, id string, score int) error
	UpdateUserFeedback(ctx context.Context, id, feedback string) error
	UpdateAudioURLs(ctx context.Context, id string, feedbackAudioURL, demoAudioURL *string) error

	// 事务支持
	WithTx(tx *gorm.DB) FeedbackRecordRepository
}

// feedbackRecordRepository 反馈记录数据库操作实现
type feedbackRecordRepository struct {
	db *gorm.DB
}

// NewFeedbackRecordRepository 创建反馈记录数据库操作实例
func NewFeedbackRecordRepository(db *gorm.DB) FeedbackRecordRepository {
	return &feedbackRecordRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *feedbackRecordRepository) WithTx(tx *gorm.DB) FeedbackRecordRepository {
	return &feedbackRecordRepository{db: tx}
}

// Create 创建反馈记录
func (r *feedbackRecordRepository) Create(ctx context.Context, record *model.FeedbackRecord) error {
	err := r.db.WithContext(ctx).Create(record).Error
	return WrapDBError(err, "create feedback record")
}

// GetByID 根据 ID 获取反馈记录
func (r *feedbackRecordRepository) GetByID(ctx context.Context, id string) (*model.FeedbackRecord, error) {
	var record model.FeedbackRecord
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&record).Error
	if err != nil {
		return nil, WrapDBError(err, "get feedback record by id")
	}
	return &record, nil
}

// GetByEvaluationID 根据评测 ID 获取反馈记录
func (r *feedbackRecordRepository) GetByEvaluationID(ctx context.Context, evaluationID string) (*model.FeedbackRecord, error) {
	var record model.FeedbackRecord
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ?", evaluationID).
		First(&record).Error
	if err != nil {
		return nil, WrapDBError(err, "get feedback record by evaluation id")
	}
	return &record, nil
}

// Update 更新反馈记录
func (r *feedbackRecordRepository) Update(ctx context.Context, record *model.FeedbackRecord) error {
	err := r.db.WithContext(ctx).Save(record).Error
	return WrapDBError(err, "update feedback record")
}

// Delete 删除反馈记录
func (r *feedbackRecordRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.FeedbackRecord{}).Error
	return WrapDBError(err, "delete feedback record")
}

// GetByStatus 根据状态分页获取反馈记录
func (r *feedbackRecordRepository) GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.FeedbackRecord, int64, error) {
	var records []*model.FeedbackRecord
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("status = ?", status).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count feedback records by status")
	}

	err = r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list feedback records by status")
	}

	return records, total, nil
}

// GetByFeedbackLevel 根据反馈级别分页获取反馈记录
func (r *feedbackRecordRepository) GetByFeedbackLevel(ctx context.Context, level string, page, pageSize int) ([]*model.FeedbackRecord, int64, error) {
	var records []*model.FeedbackRecord
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("feedback_level = ?", level).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count feedback records by level")
	}

	err = r.db.WithContext(ctx).
		Where("feedback_level = ?", level).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list feedback records by level")
	}

	return records, total, nil
}

// GetByGenerationMethod 根据生成方法分页获取反馈记录
func (r *feedbackRecordRepository) GetByGenerationMethod(ctx context.Context, method string, page, pageSize int) ([]*model.FeedbackRecord, int64, error) {
	var records []*model.FeedbackRecord
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("generation_method = ?", method).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count feedback records by method")
	}

	err = r.db.WithContext(ctx).
		Where("generation_method = ?", method).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list feedback records by method")
	}

	return records, total, nil
}

// Count 统计反馈记录总数
func (r *feedbackRecordRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count feedback records")
	}
	return count, nil
}

// CountByStatus 根据状态统计反馈记录数
func (r *feedbackRecordRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("status = ?", status).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count feedback records by status")
	}
	return count, nil
}

// CountByFeedbackLevel 根据反馈级别统计反馈记录数
func (r *feedbackRecordRepository) CountByFeedbackLevel(ctx context.Context, level string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("feedback_level = ?", level).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count feedback records by level")
	}
	return count, nil
}

// GetAverageGenerationDuration 获取平均生成时长
func (r *feedbackRecordRepository) GetAverageGenerationDuration(ctx context.Context) (float64, error) {
	var avgDuration float64
	// 只计算最近 30 天的数据
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("status = ? AND generation_duration_ms IS NOT NULL AND created_at >= ?", "completed", thirtyDaysAgo).
		Select("COALESCE(AVG(generation_duration_ms), 0)").
		Scan(&avgDuration).Error
	if err != nil {
		return 0, WrapDBError(err, "get average generation duration")
	}
	return avgDuration, nil
}

// GetAverageQualityScore 获取平均质量评分
func (r *feedbackRecordRepository) GetAverageQualityScore(ctx context.Context) (float64, error) {
	var avgScore float64
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("quality_score IS NOT NULL").
		Select("COALESCE(AVG(quality_score), 0)").
		Scan(&avgScore).Error
	if err != nil {
		return 0, WrapDBError(err, "get average quality score")
	}
	return avgScore, nil
}

// UpdateStatus 更新反馈记录状态
func (r *feedbackRecordRepository) UpdateStatus(ctx context.Context, id, status string) error {
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("id = ?", id).
		Update("status", status).Error
	return WrapDBError(err, "update feedback record status")
}

// UpdateQualityScore 更新质量评分
func (r *feedbackRecordRepository) UpdateQualityScore(ctx context.Context, id string, score int) error {
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("id = ?", id).
		Update("quality_score", score).Error
	return WrapDBError(err, "update feedback record quality score")
}

// UpdateUserFeedback 更新用户反馈
func (r *feedbackRecordRepository) UpdateUserFeedback(ctx context.Context, id, feedback string) error {
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("id = ?", id).
		Update("user_feedback", feedback).Error
	return WrapDBError(err, "update user feedback")
}

// UpdateAudioURLs 更新音频 URL
func (r *feedbackRecordRepository) UpdateAudioURLs(ctx context.Context, id string, feedbackAudioURL, demoAudioURL *string) error {
	updates := make(map[string]interface{})
	if feedbackAudioURL != nil {
		updates["feedback_audio_url"] = *feedbackAudioURL
	}
	if demoAudioURL != nil {
		updates["demo_audio_url"] = *demoAudioURL
	}
	if len(updates) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Model(&model.FeedbackRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
	return WrapDBError(err, "update feedback record audio urls")
}
