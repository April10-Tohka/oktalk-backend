// Package db 提供学习报告数据库操作
package db

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// HardWord 难词统计（周报）
type HardWord struct {
	Word  string
	Count int
}

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

	// FindLatestByUserAndPeriod 查询同一用户、同一报告类型、同一周期内最近一次报告
	// 若不存在返回 nil, nil（不报错）
	FindLatestByUserAndPeriod(ctx context.Context, userID, reportType string, periodStart, periodEnd time.Time) (*model.LearningReport, error)
	// UpdateTaskID 更新报告的 TaskID
	UpdateTaskID(ctx context.Context, reportID string, taskID string) error

	// FindByID 根据 ID 查询，不存在返回 nil, nil
	FindByID(ctx context.Context, reportID string) (*model.LearningReport, error)
	// ListByUserID 用户最新报告列表（is_latest=true），按 period_start_date 降序
	ListByUserID(ctx context.Context, userID string) ([]*model.LearningReport, error)
	// UpdateContent 更新报告 JSON 内容与任务 ID
	UpdateContent(ctx context.Context, reportID string, content string, taskID string) error
	// MarkOldReportsNotLatest 同周期内除 currentID 外置为非最新
	MarkOldReportsNotLatest(ctx context.Context, userID, reportType string, periodStart time.Time, currentID string) error

	// ---- 周报统计（Worker）----
	CountEvaluations(ctx context.Context, userID string, start, end time.Time) (int, error)
	CountConversations(ctx context.Context, userID string, start, end time.Time) (int, error)
	CountPersistenceDays(ctx context.Context, userID string, start, end time.Time) (int, error)
	GetRadarScores(ctx context.Context, userID string, start, end time.Time) (avgRaw, avgFluency, avgIntegrity float32, err error)
	GetHardWords(ctx context.Context, userID string, start, end time.Time) ([]HardWord, error)

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

// FindLatestByUserAndPeriod 查询同一用户、同一报告类型、同一周期内最近一次报告
func (r *learningReportRepository) FindLatestByUserAndPeriod(ctx context.Context, userID, reportType string, periodStart, periodEnd time.Time) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND report_type = ? AND period_start_date = ? AND period_end_date = ?", userID, reportType, periodStart, periodEnd).
		Order("created_at DESC").
		First(&report).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 查无结果不报错
		}
		return nil, WrapDBError(err, "find latest learning report by period")
	}
	return &report, nil
}

// UpdateTaskID 更新报告的 TaskID
func (r *learningReportRepository) UpdateTaskID(ctx context.Context, reportID string, taskID string) error {
	err := r.db.WithContext(ctx).
		Model(&model.LearningReport{}).
		Where("id = ?", reportID).
		Update("task_id", taskID).Error
	return WrapDBError(err, "update learning report task_id")
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

// FindByID 根据 ID 查询，不存在返回 nil, nil
func (r *learningReportRepository) FindByID(ctx context.Context, reportID string) (*model.LearningReport, error) {
	var report model.LearningReport
	err := r.db.WithContext(ctx).Where("id = ?", reportID).First(&report).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, WrapDBError(err, "find learning report by id")
	}
	return &report, nil
}

// ListByUserID 查询 is_latest=true 的报告
func (r *learningReportRepository) ListByUserID(ctx context.Context, userID string) ([]*model.LearningReport, error) {
	var list []*model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_latest = ?", userID, true).
		Order("period_start_date DESC").
		Find(&list).Error
	if err != nil {
		return nil, WrapDBError(err, "list learning reports by user")
	}
	return list, nil
}

// UpdateContent 更新 content 与 task_id
func (r *learningReportRepository) UpdateContent(ctx context.Context, reportID string, content string, taskID string) error {
	err := r.db.WithContext(ctx).Model(&model.LearningReport{}).
		Where("id = ?", reportID).
		Updates(map[string]interface{}{
			"content": content,
			"task_id": taskID,
		}).Error
	return WrapDBError(err, "update learning report content")
}

// MarkOldReportsNotLatest 同周期旧报告置为非最新
func (r *learningReportRepository) MarkOldReportsNotLatest(ctx context.Context, userID, reportType string, periodStart time.Time, currentID string) error {
	err := r.db.WithContext(ctx).Model(&model.LearningReport{}).
		Where("user_id = ? AND report_type = ? AND period_start_date = ? AND id != ?",
			userID, reportType, periodStart, currentID).
		Update("is_latest", false).Error
	return WrapDBError(err, "mark old reports not latest")
}

// CountEvaluations 统计 pronunciation_records 条数
func (r *learningReportRepository) CountEvaluations(ctx context.Context, userID string, start, end time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.PronunciationRecord{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Count(&n).Error
	return int(n), WrapDBError(err, "count evaluations")
}

// CountConversations 统计 scene_messages 条数
func (r *learningReportRepository) CountConversations(ctx context.Context, userID string, start, end time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.SceneMessage{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Count(&n).Error
	return int(n), WrapDBError(err, "count scene messages")
}

// CountPersistenceDays 两表练习日期并集天数
func (r *learningReportRepository) CountPersistenceDays(ctx context.Context, userID string, start, end time.Time) (int, error) {
	var n int64
	sql := `
SELECT COUNT(DISTINCT d) FROM (
  SELECT DATE(created_at) AS d FROM pronunciation_records
    WHERE user_id = ? AND created_at >= ? AND created_at <= ?
  UNION
  SELECT DATE(created_at) AS d FROM scene_messages
    WHERE user_id = ? AND created_at >= ? AND created_at <= ?
) t`
	err := r.db.WithContext(ctx).Raw(sql, userID, start, end, userID, start, end).Scan(&n).Error
	return int(n), WrapDBError(err, "count persistence days")
}

// GetRadarScores 发音记录三项均值
func (r *learningReportRepository) GetRadarScores(ctx context.Context, userID string, start, end time.Time) (avgRaw, avgFluency, avgIntegrity float32, err error) {
	var row struct {
		AvgRaw       float64 `gorm:"column:avg_raw"`
		AvgFluency   float64 `gorm:"column:avg_fluency"`
		AvgIntegrity float64 `gorm:"column:avg_integrity"`
	}
	err = r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(AVG(total_score), 0) AS avg_raw,
			COALESCE(AVG(fluency_score), 0) AS avg_fluency,
			COALESCE(AVG(integrity_score), 0) AS avg_integrity
		FROM pronunciation_records
		WHERE user_id = ? AND created_at >= ? AND created_at <= ?`,
		userID, start, end).Scan(&row).Error
	if err != nil {
		return 0, 0, 0, WrapDBError(err, "get radar scores")
	}
	return float32(row.AvgRaw), float32(row.AvgFluency), float32(row.AvgIntegrity), nil
}

// GetHardWords 本周难词 Top4
func (r *learningReportRepository) GetHardWords(ctx context.Context, userID string, start, end time.Time) ([]HardWord, error) {
	var rows []struct {
		PW string `gorm:"column:problem_words"`
	}
	err := r.db.WithContext(ctx).Model(&model.PronunciationRecord{}).
		Select("problem_words").
		Where("user_id = ? AND created_at >= ? AND created_at <= ? AND problem_words IS NOT NULL AND problem_words != '' AND problem_words != 'null'",
			userID, start, end).
		Find(&rows).Error
	if err != nil {
		return nil, WrapDBError(err, "get hard words rows")
	}
	freq := make(map[string]int)
	for _, row := range rows {
		var words []string
		if err := json.Unmarshal([]byte(row.PW), &words); err != nil {
			continue
		}
		for _, w := range words {
			w = strings.TrimSpace(strings.ToLower(w))
			if w == "" {
				continue
			}
			freq[w]++
		}
	}
	type pair struct {
		w string
		c int
	}
	var plist []pair
	for w, c := range freq {
		plist = append(plist, pair{w, c})
	}
	sort.Slice(plist, func(i, j int) bool {
		if plist[i].c == plist[j].c {
			return plist[i].w < plist[j].w
		}
		return plist[i].c > plist[j].c
	})
	out := make([]HardWord, 0, 4)
	for i := 0; i < len(plist) && i < 4; i++ {
		out = append(out, HardWord{Word: plist[i].w, Count: plist[i].c})
	}
	return out, nil
}
