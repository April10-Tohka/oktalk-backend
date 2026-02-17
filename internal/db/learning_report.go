// Package db 提供学习报告数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// LearningReportRepository 学习报告数据库操作接口
type LearningReportRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, report *model.LearningReport) error
	GetByID(ctx context.Context, id string) (*model.LearningReport, error)
	Update(ctx context.Context, report *model.LearningReport) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.LearningReport, int64, error)
	GetByUserIDAndType(ctx context.Context, userID, reportType string, page, pageSize int) ([]*model.LearningReport, int64, error)
	GetByUserIDAndPeriod(ctx context.Context, userID string, start, end time.Time) (*model.LearningReport, error)
	GetLatestByUserID(ctx context.Context, userID string) (*model.LearningReport, error)
	GetLatestByUserIDAndType(ctx context.Context, userID, reportType string) (*model.LearningReport, error)

	// 统计方法
	Count(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CountByType(ctx context.Context, reportType string) (int64, error)

	// 预加载方法
	GetWithUser(ctx context.Context, id string) (*model.LearningReport, error)

	// 事务支持
	WithTx(tx *gorm.DB) LearningReportRepository
}

// learningReportRepository 学习报告数据库操作实现
type learningReportRepository struct {
	db *gorm.DB
}

// NewLearningReportRepository 创建学习报告数据库操作实例
func NewLearningReportRepository(db *gorm.DB) LearningReportRepository {
	return &learningReportRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *learningReportRepository) WithTx(tx *gorm.DB) LearningReportRepository {
	return &learningReportRepository{db: tx}
}

// Create 创建学习报告
func (r *learningReportRepository) Create(ctx context.Context, report *model.LearningReport) error {
	err := r.db.WithContext(ctx).Create(report).Error
	return WrapDBError(err, "create learning report")
}

// GetByID 根据 ID 获取学习报告
func (r *learningReportRepository) GetByID(ctx context.Context, id string) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&report).Error
	if err != nil {
		return nil, WrapDBError(err, "get learning report by id")
	}
	return &report, nil
}

// Update 更新学习报告
func (r *learningReportRepository) Update(ctx context.Context, report *model.LearningReport) error {
	err := r.db.WithContext(ctx).Save(report).Error
	return WrapDBError(err, "update learning report")
}

// Delete 删除学习报告
func (r *learningReportRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.LearningReport{}).Error
	return WrapDBError(err, "delete learning report")
}

// GetByUserID 根据用户 ID 分页获取学习报告
func (r *learningReportRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.LearningReport, int64, error) {
	var reports []*model.LearningReport
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count learning reports by user id")
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("period_start_date DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&reports).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list learning reports by user id")
	}

	return reports, total, nil
}

// GetByUserIDAndType 根据用户 ID 和报告类型分页获取学习报告
func (r *learningReportRepository) GetByUserIDAndType(ctx context.Context, userID, reportType string, page, pageSize int) ([]*model.LearningReport, int64, error) {
	var reports []*model.LearningReport
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Where("user_id = ? AND report_type = ?", userID, reportType).
		Count(&total).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "count learning reports by type")
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ? AND report_type = ?", userID, reportType).
		Order("period_start_date DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&reports).Error
	if err != nil {
		return nil, 0, WrapDBError(err, "list learning reports by type")
	}

	return reports, total, nil
}

// GetByUserIDAndPeriod 根据用户 ID 和时间周期获取学习报告
func (r *learningReportRepository) GetByUserIDAndPeriod(ctx context.Context, userID string, start, end time.Time) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND period_start_date = ? AND period_end_date = ?", userID, start, end).
		First(&report).Error
	if err != nil {
		return nil, WrapDBError(err, "get learning report by period")
	}
	return &report, nil
}

// GetLatestByUserID 获取用户最新的学习报告
func (r *learningReportRepository) GetLatestByUserID(ctx context.Context, userID string) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&report).Error
	if err != nil {
		return nil, WrapDBError(err, "get latest learning report")
	}
	return &report, nil
}

// GetLatestByUserIDAndType 获取用户最新的指定类型学习报告
func (r *learningReportRepository) GetLatestByUserIDAndType(ctx context.Context, userID, reportType string) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND report_type = ?", userID, reportType).
		Order("created_at DESC").
		First(&report).Error
	if err != nil {
		return nil, WrapDBError(err, "get latest learning report by type")
	}
	return &report, nil
}

// Count 统计学习报告总数
func (r *learningReportRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count learning reports")
	}
	return count, nil
}

// CountByUserID 统计用户学习报告数
func (r *learningReportRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count learning reports by user id")
	}
	return count, nil
}

// CountByType 统计指定类型的学习报告数
func (r *learningReportRepository) CountByType(ctx context.Context, reportType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Where("report_type = ?", reportType).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count learning reports by type")
	}
	return count, nil
}

// GetWithUser 获取学习报告及其关联用户
func (r *learningReportRepository) GetWithUser(ctx context.Context, id string) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&report).Error
	if err != nil {
		return nil, WrapDBError(err, "get learning report with user")
	}
	return &report, nil
}
