// Package db 提供报告统计明细数据库操作
package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// ReportStatisticRepository 报告统计明细数据库操作接口
type ReportStatisticRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, statistic *model.ReportStatistic) error
	GetByID(ctx context.Context, id string) (*model.ReportStatistic, error)
	Update(ctx context.Context, statistic *model.ReportStatistic) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByReportID(ctx context.Context, reportID string) ([]*model.ReportStatistic, error)
	GetByReportIDAndDate(ctx context.Context, reportID string, date time.Time) (*model.ReportStatistic, error)
	GetByDateRange(ctx context.Context, reportID string, start, end time.Time) ([]*model.ReportStatistic, error)

	// 统计方法
	CountByReportID(ctx context.Context, reportID string) (int64, error)
	GetTotalStudyMinutes(ctx context.Context, reportID string) (int, error)
	GetAverageDailyEvalScore(ctx context.Context, reportID string) (float64, error)

	// 批量操作
	BatchCreate(ctx context.Context, statistics []*model.ReportStatistic) error
	DeleteByReportID(ctx context.Context, reportID string) error

	// 事务支持
	WithTx(tx *gorm.DB) ReportStatisticRepository
}

// reportStatisticRepository 报告统计明细数据库操作实现
type reportStatisticRepository struct {
	db *gorm.DB
}

// NewReportStatisticRepository 创建报告统计明细数据库操作实例
func NewReportStatisticRepository(db *gorm.DB) ReportStatisticRepository {
	return &reportStatisticRepository{db: db}
}

// WithTx 返回使用事务的 Repository
func (r *reportStatisticRepository) WithTx(tx *gorm.DB) ReportStatisticRepository {
	return &reportStatisticRepository{db: tx}
}

// Create 创建报告统计明细
func (r *reportStatisticRepository) Create(ctx context.Context, statistic *model.ReportStatistic) error {
	err := r.db.WithContext(ctx).Create(statistic).Error
	return WrapDBError(err, "create report statistic")
}

// GetByID 根据 ID 获取报告统计明细
func (r *reportStatisticRepository) GetByID(ctx context.Context, id string) (*model.ReportStatistic, error) {
	var statistic model.ReportStatistic
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&statistic).Error
	if err != nil {
		return nil, WrapDBError(err, "get report statistic by id")
	}
	return &statistic, nil
}

// Update 更新报告统计明细
func (r *reportStatisticRepository) Update(ctx context.Context, statistic *model.ReportStatistic) error {
	err := r.db.WithContext(ctx).Save(statistic).Error
	return WrapDBError(err, "update report statistic")
}

// Delete 删除报告统计明细
func (r *reportStatisticRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.ReportStatistic{}).Error
	return WrapDBError(err, "delete report statistic")
}

// GetByReportID 获取报告的所有统计明细
func (r *reportStatisticRepository) GetByReportID(ctx context.Context, reportID string) ([]*model.ReportStatistic, error) {
	var statistics []*model.ReportStatistic
	err := r.db.WithContext(ctx).
		Where("report_id = ?", reportID).
		Order("stat_date ASC").
		Find(&statistics).Error
	if err != nil {
		return nil, WrapDBError(err, "get report statistics by report id")
	}
	return statistics, nil
}

// GetByReportIDAndDate 根据报告 ID 和日期获取统计明细
func (r *reportStatisticRepository) GetByReportIDAndDate(ctx context.Context, reportID string, date time.Time) (*model.ReportStatistic, error) {
	var statistic model.ReportStatistic
	err := r.db.WithContext(ctx).
		Where("report_id = ? AND stat_date = ?", reportID, date).
		First(&statistic).Error
	if err != nil {
		return nil, WrapDBError(err, "get report statistic by date")
	}
	return &statistic, nil
}

// GetByDateRange 获取指定日期范围的统计明细
func (r *reportStatisticRepository) GetByDateRange(ctx context.Context, reportID string, start, end time.Time) ([]*model.ReportStatistic, error) {
	var statistics []*model.ReportStatistic
	err := r.db.WithContext(ctx).
		Where("report_id = ? AND stat_date >= ? AND stat_date <= ?", reportID, start, end).
		Order("stat_date ASC").
		Find(&statistics).Error
	if err != nil {
		return nil, WrapDBError(err, "get report statistics by date range")
	}
	return statistics, nil
}

// CountByReportID 统计报告的统计明细数
func (r *reportStatisticRepository) CountByReportID(ctx context.Context, reportID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ReportStatistic{}).
		Where("report_id = ?", reportID).
		Count(&count).Error
	if err != nil {
		return 0, WrapDBError(err, "count report statistics")
	}
	return count, nil
}

// GetTotalStudyMinutes 获取报告周期内的总学习时长
func (r *reportStatisticRepository) GetTotalStudyMinutes(ctx context.Context, reportID string) (int, error) {
	var total int
	err := r.db.WithContext(ctx).
		Model(&model.ReportStatistic{}).
		Where("report_id = ?", reportID).
		Select("COALESCE(SUM(daily_study_minutes), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, WrapDBError(err, "get total study minutes")
	}
	return total, nil
}

// GetAverageDailyEvalScore 获取报告周期内的平均每日评测分数
func (r *reportStatisticRepository) GetAverageDailyEvalScore(ctx context.Context, reportID string) (float64, error) {
	var avgScore float64
	err := r.db.WithContext(ctx).
		Model(&model.ReportStatistic{}).
		Where("report_id = ? AND daily_evaluations > 0", reportID).
		Select("COALESCE(AVG(daily_avg_eval_score), 0)").
		Scan(&avgScore).Error
	if err != nil {
		return 0, WrapDBError(err, "get average daily eval score")
	}
	return avgScore, nil
}

// BatchCreate 批量创建报告统计明细
func (r *reportStatisticRepository) BatchCreate(ctx context.Context, statistics []*model.ReportStatistic) error {
	if len(statistics) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(statistics, 100).Error
	return WrapDBError(err, "batch create report statistics")
}

// DeleteByReportID 删除报告的所有统计明细
func (r *reportStatisticRepository) DeleteByReportID(ctx context.Context, reportID string) error {
	err := r.db.WithContext(ctx).
		Where("report_id = ?", reportID).
		Delete(&model.ReportStatistic{}).Error
	return WrapDBError(err, "delete report statistics by report id")
}
