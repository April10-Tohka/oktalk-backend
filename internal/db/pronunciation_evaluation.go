// Package db 提供发音评测数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// PronunciationEvaluationRepository 发音评测数据库操作接口
type PronunciationEvaluationRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, evaluation *model.PronunciationEvaluation) error
	GetByID(ctx context.Context, id string) (*model.PronunciationEvaluation, error)
	Update(ctx context.Context, evaluation *model.PronunciationEvaluation) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error)
	GetByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error)
	GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error)
	GetByFeedbackLevel(ctx context.Context, level string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error)

	// 统计方法
	Count(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CountByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (int64, error)
	CountByFeedbackLevel(ctx context.Context, userID, level string) (int64, error)
	GetAverageScoreByUserID(ctx context.Context, userID string) (float64, error)
	GetAverageScoreByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (float64, error)

	// 更新方法
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateFeedback(ctx context.Context, id string, level, text string, audioURL *string) error
	UpdateScores(ctx context.Context, id string, overall, accuracy, fluency, integrity int) error

	// 预加载方法
	GetWithDetails(ctx context.Context, id string) (*model.PronunciationEvaluation, error)
	GetWithFeedbackRecord(ctx context.Context, id string) (*model.PronunciationEvaluation, error)

	// 事务支持
	WithTx(tx *gorm.DB) PronunciationEvaluationRepository
}

// pronunciationEvaluationRepository 发音评测数据库操作实现
type pronunciationEvaluationRepository struct {
	db *gorm.DB
}

// NewPronunciationEvaluationRepository 创建发音评测数据库操作实例
func NewPronunciationEvaluationRepository(db *gorm.DB) PronunciationEvaluationRepository {
	return &pronunciationEvaluationRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *pronunciationEvaluationRepository) WithTx(tx *gorm.DB) PronunciationEvaluationRepository {
	return &pronunciationEvaluationRepository{db: tx}
}

// Create 创建发音评测记录
func (r *pronunciationEvaluationRepository) Create(ctx context.Context, evaluation *model.PronunciationEvaluation) error {
	err := r.db.WithContext(ctx).Create(evaluation).Error
	return WrapDBError(err, "create pronunciation evaluation")
}

// GetByID 根据 ID 获取发音评测记录
func (r *pronunciationEvaluationRepository) GetByID(ctx context.Context, id string) (*model.PronunciationEvaluation, error) {
	var evaluation model.PronunciationEvaluation
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&evaluation).Error
	if err != nil {
		return nil, WrapDBError(err, "get pronunciation evaluation by id")
	}
	return &evaluation, nil
}

// Update 更新发音评测记录
func (r *pronunciationEvaluationRepository) Update(ctx context.Context, evaluation *model.PronunciationEvaluation) error {
	err := r.db.WithContext(ctx).Save(evaluation).Error
	return WrapDBError(err, "update pronunciation evaluation")
}

// Delete 删除发音评测记录
func (r *pronunciationEvaluationRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.PronunciationEvaluation{}).Error
	return WrapDBError(err, "delete pronunciation evaluation")
}

// GetByUserID 根据用户 ID 分页获取评测列表
func (r *pronunciationEvaluationRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error) {
	var evaluations []*model.PronunciationEvaluation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count pronunciation evaluations by user id")
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&evaluations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list pronunciation evaluations by user id")
	}

	return evaluations, total, nil
}

// GetByUserIDAndDateRange 根据用户 ID 和时间范围分页获取评测列表
func (r *pronunciationEvaluationRepository) GetByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error) {
	var evaluations []*model.PronunciationEvaluation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := r.db.WithContext(ctx).Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count pronunciation evaluations by date range")
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&evaluations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list pronunciation evaluations by date range")
	}

	return evaluations, total, nil
}

// GetByStatus 根据状态分页获取评测列表
func (r *pronunciationEvaluationRepository) GetByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error) {
	var evaluations []*model.PronunciationEvaluation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("status = ?", status).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count pronunciation evaluations by status")
	}

	err = r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&evaluations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list pronunciation evaluations by status")
	}

	return evaluations, total, nil
}

// GetByFeedbackLevel 根据反馈级别分页获取评测列表
func (r *pronunciationEvaluationRepository) GetByFeedbackLevel(ctx context.Context, level string, page, pageSize int) ([]*model.PronunciationEvaluation, int64, error) {
	var evaluations []*model.PronunciationEvaluation
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("feedback_level = ?", level).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count pronunciation evaluations by feedback level")
	}

	err = r.db.WithContext(ctx).
		Where("feedback_level = ?", level).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&evaluations).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list pronunciation evaluations by feedback level")
	}

	return evaluations, total, nil
}

// Count 统计评测总数
func (r *pronunciationEvaluationRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count pronunciation evaluations")
	}
	return count, nil
}

// CountByUserID 统计用户评测数
func (r *pronunciationEvaluationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count pronunciation evaluations by user id")
	}
	return count, nil
}

// CountByUserIDAndDateRange 统计用户在指定时间范围内的评测数
func (r *pronunciationEvaluationRepository) CountByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count pronunciation evaluations by date range")
	}
	return count, nil
}

// CountByFeedbackLevel 统计用户指定反馈级别的评测数
func (r *pronunciationEvaluationRepository) CountByFeedbackLevel(ctx context.Context, userID, level string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND feedback_level = ?", userID, level).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count pronunciation evaluations by feedback level")
	}
	return count, nil
}

// GetAverageScoreByUserID 获取用户平均评分
func (r *pronunciationEvaluationRepository) GetAverageScoreByUserID(ctx context.Context, userID string) (float64, error) {
	var avgScore float64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND status = ?", userID, model.EvaluationStatusCompleted).
		Select("COALESCE(AVG(overall_score), 0)").
		Scan(&avgScore).Error
	if err != nil {
		return 0, WrapDBError(err, "get average score by user id")
	}
	return avgScore, nil
}

// GetAverageScoreByUserIDAndDateRange 获取用户在指定时间范围内的平均评分
func (r *pronunciationEvaluationRepository) GetAverageScoreByUserIDAndDateRange(ctx context.Context, userID string, start, end time.Time) (float64, error) {
	var avgScore float64
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND status = ? AND created_at >= ? AND created_at < ?", userID, model.EvaluationStatusCompleted, start, end).
		Select("COALESCE(AVG(overall_score), 0)").
		Scan(&avgScore).Error
	if err != nil {
		return 0, WrapDBError(err, "get average score by date range")
	}
	return avgScore, nil
}

// UpdateStatus 更新评测状态
func (r *pronunciationEvaluationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("id = ?", id).
		Update("status", status).Error
	return WrapDBError(err, "update pronunciation evaluation status")
}

// UpdateFeedback 更新反馈信息
func (r *pronunciationEvaluationRepository) UpdateFeedback(ctx context.Context, id string, level, text string, audioURL *string) error {
	updates := map[string]interface{}{
		"feedback_level":     level,
		"feedback_text":      text,
		"feedback_audio_url": audioURL,
	}
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("id = ?", id).
		Updates(updates).Error
	return WrapDBError(err, "update pronunciation evaluation feedback")
}

// UpdateScores 更新评分
func (r *pronunciationEvaluationRepository) UpdateScores(ctx context.Context, id string, overall, accuracy, fluency, integrity int) error {
	updates := map[string]interface{}{
		"overall_score":   overall,
		"accuracy_score":  accuracy,
		"fluency_score":   fluency,
		"integrity_score": integrity,
	}
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("id = ?", id).
		Updates(updates).Error
	return WrapDBError(err, "update pronunciation evaluation scores")
}

// GetWithDetails 获取评测记录及其详细数据
func (r *pronunciationEvaluationRepository) GetWithDetails(ctx context.Context, id string) (*model.PronunciationEvaluation, error) {
	var evaluation model.PronunciationEvaluation
	err := r.db.WithContext(ctx).
		Preload("EvaluationDetails", func(db *gorm.DB) *gorm.DB {
			return db.Order("word_index ASC")
		}).
		Where("id = ?", id).
		First(&evaluation).Error
	if err != nil {
		return nil, WrapDBError(err, "get pronunciation evaluation with details")
	}
	return &evaluation, nil
}

// GetWithFeedbackRecord 获取评测记录及其反馈记录
func (r *pronunciationEvaluationRepository) GetWithFeedbackRecord(ctx context.Context, id string) (*model.PronunciationEvaluation, error) {
	var evaluation model.PronunciationEvaluation
	err := r.db.WithContext(ctx).
		Preload("FeedbackRecord").
		Where("id = ?", id).
		First(&evaluation).Error
	if err != nil {
		return nil, WrapDBError(err, "get pronunciation evaluation with feedback record")
	}
	return &evaluation, nil
}
