// Package db 提供评测详情数据库操作
package db

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// EvaluationDetailRepository 评测详情数据库操作接口
type EvaluationDetailRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, detail *model.EvaluationDetail) error
	GetByID(ctx context.Context, id string) (*model.EvaluationDetail, error)
	Update(ctx context.Context, detail *model.EvaluationDetail) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByEvaluationID(ctx context.Context, evaluationID string) ([]*model.EvaluationDetail, error)
	GetProblemWords(ctx context.Context, evaluationID string) ([]*model.EvaluationDetail, error)
	GetByWordScore(ctx context.Context, evaluationID string, minScore, maxScore int) ([]*model.EvaluationDetail, error)

	// 统计方法
	CountByEvaluationID(ctx context.Context, evaluationID string) (int64, error)
	CountProblemWords(ctx context.Context, evaluationID string) (int64, error)
	GetAverageWordScore(ctx context.Context, evaluationID string) (float64, error)

	// 批量操作
	BatchCreate(ctx context.Context, details []*model.EvaluationDetail) error
	DeleteByEvaluationID(ctx context.Context, evaluationID string) error

	// 事务支持
	WithTx(tx *gorm.DB) EvaluationDetailRepository
}

// evaluationDetailRepository 评测详情数据库操作实现
type evaluationDetailRepository struct {
	db *gorm.DB
}

// NewEvaluationDetailRepository 创建评测详情数据库操作实例
func NewEvaluationDetailRepository(db *gorm.DB) EvaluationDetailRepository {
	return &evaluationDetailRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *evaluationDetailRepository) WithTx(tx *gorm.DB) EvaluationDetailRepository {
	return &evaluationDetailRepository{db: tx}
}

// Create 创建评测详情
func (r *evaluationDetailRepository) Create(ctx context.Context, detail *model.EvaluationDetail) error {
	err := r.db.WithContext(ctx).Create(detail).Error
	return WrapDBError(err, "create evaluation detail")
}

// GetByID 根据 ID 获取评测详情
func (r *evaluationDetailRepository) GetByID(ctx context.Context, id string) (*model.EvaluationDetail, error) {
	var detail model.EvaluationDetail
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&detail).Error
	if err != nil {
		return nil, WrapDBError(err, "get evaluation detail by id")
	}
	return &detail, nil
}

// Update 更新评测详情
func (r *evaluationDetailRepository) Update(ctx context.Context, detail *model.EvaluationDetail) error {
	err := r.db.WithContext(ctx).Save(detail).Error
	return WrapDBError(err, "update evaluation detail")
}

// Delete 删除评测详情
func (r *evaluationDetailRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.EvaluationDetail{}).Error
	return WrapDBError(err, "delete evaluation detail")
}

// GetByEvaluationID 获取评测的所有详情
func (r *evaluationDetailRepository) GetByEvaluationID(ctx context.Context, evaluationID string) ([]*model.EvaluationDetail, error) {
	var details []*model.EvaluationDetail
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ?", evaluationID).
		Order("word_index ASC").
		Find(&details).Error
	if err != nil {
		return nil, WrapDBError(err, "get evaluation details by evaluation id")
	}
	return details, nil
}

// GetProblemWords 获取问题单词（评分低于阈值的单词）
func (r *evaluationDetailRepository) GetProblemWords(ctx context.Context, evaluationID string) ([]*model.EvaluationDetail, error) {
	var details []*model.EvaluationDetail
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ? AND is_problem_word = ?", evaluationID, true).
		Order("word_score ASC").
		Find(&details).Error
	if err != nil {
		return nil, WrapDBError(err, "get problem words")
	}
	return details, nil
}

// GetByWordScore 根据单词评分范围获取详情
func (r *evaluationDetailRepository) GetByWordScore(ctx context.Context, evaluationID string, minScore, maxScore int) ([]*model.EvaluationDetail, error) {
	var details []*model.EvaluationDetail
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ? AND word_score >= ? AND word_score <= ?", evaluationID, minScore, maxScore).
		Order("word_index ASC").
		Find(&details).Error
	if err != nil {
		return nil, WrapDBError(err, "get evaluation details by word score")
	}
	return details, nil
}

// CountByEvaluationID 统计评测详情数
func (r *evaluationDetailRepository) CountByEvaluationID(ctx context.Context, evaluationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.EvaluationDetail{}).
		Where("evaluation_id = ?", evaluationID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count evaluation details")
	}
	return count, nil
}

// CountProblemWords 统计问题单词数
func (r *evaluationDetailRepository) CountProblemWords(ctx context.Context, evaluationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.EvaluationDetail{}).
		Where("evaluation_id = ? AND is_problem_word = ?", evaluationID, true).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count problem words")
	}
	return count, nil
}

// GetAverageWordScore 获取单词平均评分
func (r *evaluationDetailRepository) GetAverageWordScore(ctx context.Context, evaluationID string) (float64, error) {
	var avgScore float64
	err := r.db.WithContext(ctx).
		Model(&model.EvaluationDetail{}).
		Where("evaluation_id = ?", evaluationID).
		Select("COALESCE(AVG(word_score), 0)").
		Scan(&avgScore).Error
	if err != nil {
		return 0, WrapDBError(err, "get average word score")
	}
	return avgScore, nil
}

// BatchCreate 批量创建评测详情
func (r *evaluationDetailRepository) BatchCreate(ctx context.Context, details []*model.EvaluationDetail) error {
	if len(details) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(details, 100).Error
	return WrapDBError(err, "batch create evaluation details")
}

// DeleteByEvaluationID 删除评测的所有详情
func (r *evaluationDetailRepository) DeleteByEvaluationID(ctx context.Context, evaluationID string) error {
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ?", evaluationID).
		Delete(&model.EvaluationDetail{}).Error
	return WrapDBError(err, "delete evaluation details by evaluation id")
}
