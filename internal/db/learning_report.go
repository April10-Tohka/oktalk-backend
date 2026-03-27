// Package db 提供学习报告数据库操作
package db

import (
	"context"
	"errors"
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

	// FindLatestByUserAndPeriod 查询同一用户、同一报告类型、同一周期内最近一次报告
	// 若不存在返回 nil, nil（不报错）
	FindLatestByUserAndPeriod(ctx context.Context, userID, reportType string, periodStart, periodEnd time.Time) (*model.LearningReport, error)
	// FindReportByID 按 ID 查询，不存在返回 nil, nil
	FindReportByID(ctx context.Context, reportID string) (*model.LearningReport, error)
	// ListLatestByUserID 用户所有 is_latest=true 的报告，按 period_start_date 降序
	ListLatestByUserID(ctx context.Context, userID string) ([]*model.LearningReport, error)
	// UpdateContent 更新 content
	UpdateContent(ctx context.Context, reportID string, content string) error
	// UpdateIsLatest 同周期（按 period_start_date）旧报告置为非最新
	UpdateIsLatest(ctx context.Context, userID string, periodStart time.Time, excludeID string) error
	// UpdateTaskID 更新任务的 TaskID（异步报告）
	UpdateTaskID(ctx context.Context, reportID string, taskID string) error

	// CountEvaluations 本周有效发音评测次数（pronunciation_records, is_rejected=false）
	CountEvaluations(ctx context.Context, userID string, start, end time.Time) (int, error)
	// CountConversations 本周场景对话消息条数
	CountConversations(ctx context.Context, userID string, start, end time.Time) (int, error)
	// CountPersistenceDays 两表 created_at 日期并集天数
	CountPersistenceDays(ctx context.Context, userID string, start, end time.Time) (int, error)
	// GetAvgScores 四维原始分 AVG（0-5）
	GetAvgScores(ctx context.Context, userID string, start, end time.Time) (accuracy, fluency, integrity, standard float64, err error)
	// GetProblemWordsList problem_words JSON 原始字符串列表
	GetProblemWordsList(ctx context.Context, userID string, start, end time.Time) ([]string, error)
	// GetSceneStats passRate 百分制整数，completedScenes 完成场景数
	GetSceneStats(ctx context.Context, userID string, start, end time.Time) (passRate int, completedScenes int, err error)

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

// FindReportByID 按 ID 查询，不存在返回 nil, nil
func (r *learningReportRepository) FindReportByID(ctx context.Context, reportID string) (*model.LearningReport, error) {
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

// ListLatestByUserID 查询用户 is_latest=true 的报告
func (r *learningReportRepository) ListLatestByUserID(ctx context.Context, userID string) ([]*model.LearningReport, error) {
	var list []*model.LearningReport
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_latest = ?", userID, true).
		Order("period_start_date DESC").
		Find(&list).Error
	if err != nil {
		return nil, WrapDBError(err, "list latest learning reports")
	}
	return list, nil
}

// UpdateContent 更新 content
func (r *learningReportRepository) UpdateContent(ctx context.Context, reportID string, content string) error {
	err := r.db.WithContext(ctx).Model(&model.LearningReport{}).
		Where("id = ?", reportID).
		Update("content", content).Error
	return WrapDBError(err, "update learning report content")
}

// UpdateIsLatest 同周期其他周报置为非最新
func (r *learningReportRepository) UpdateIsLatest(ctx context.Context, userID string, periodStart time.Time, excludeID string) error {
	err := r.db.WithContext(ctx).Model(&model.LearningReport{}).
		Where("user_id = ? AND report_type = ? AND DATE(period_start_date) = DATE(?) AND id != ?",
			userID, "weekly", periodStart, excludeID).
		Update("is_latest", false).Error
	return WrapDBError(err, "update learning report is_latest")
}

// CountEvaluations 有效发音评测次数
func (r *learningReportRepository) CountEvaluations(ctx context.Context, userID string, start, end time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.PronunciationRecord{}).
		Where("user_id = ? AND is_rejected = ? AND created_at >= ? AND created_at <= ?", userID, false, start, end).
		Count(&n).Error
	return int(n), WrapDBError(err, "count evaluations")
}

// CountConversations 场景消息条数
func (r *learningReportRepository) CountConversations(ctx context.Context, userID string, start, end time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.SceneMessage{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Count(&n).Error
	return int(n), WrapDBError(err, "count scene messages")
}

// CountPersistenceDays 两表日期并集天数
func (r *learningReportRepository) CountPersistenceDays(ctx context.Context, userID string, start, end time.Time) (int, error) {
	type row struct {
		N int `gorm:"column:n"`
	}
	var out row
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT d) AS n FROM (
			SELECT DATE(created_at) AS d FROM pronunciation_records
			WHERE user_id = ? AND created_at >= ? AND created_at <= ?
			UNION
			SELECT DATE(created_at) AS d FROM scene_messages
			WHERE user_id = ? AND created_at >= ? AND created_at <= ?
		) t`,
		userID, start, end, userID, start, end).Scan(&out).Error
	if err != nil {
		return 0, WrapDBError(err, "count persistence days")
	}
	return out.N, nil
}

// GetAvgScores 四维平均分（原始 0-5）
func (r *learningReportRepository) GetAvgScores(ctx context.Context, userID string, start, end time.Time) (float64, float64, float64, float64, error) {
	type agg struct {
		AvgA float64 `gorm:"column:avg_a"`
		AvgF float64 `gorm:"column:avg_f"`
		AvgI float64 `gorm:"column:avg_i"`
		AvgS float64 `gorm:"column:avg_s"`
	}
	var a agg
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(AVG(accuracy_score), 0) AS avg_a,
			COALESCE(AVG(fluency), 0) AS avg_f,
			COALESCE(AVG(integrity), 0) AS avg_i,
			COALESCE(AVG(standard_score), 0) AS avg_s
		FROM pronunciation_records
		WHERE user_id = ? AND is_rejected = false AND created_at >= ? AND created_at <= ?`,
		userID, start, end).Scan(&a).Error
	if err != nil {
		return 0, 0, 0, 0, WrapDBError(err, "get avg scores")
	}
	return a.AvgA, a.AvgF, a.AvgI, a.AvgS, nil
}

// GetProblemWordsList 拉取 problem_words 列原始 JSON 字符串
func (r *learningReportRepository) GetProblemWordsList(ctx context.Context, userID string, start, end time.Time) ([]string, error) {
	var rows []struct {
		PW string `gorm:"column:problem_words"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT problem_words FROM pronunciation_records
		WHERE user_id = ? AND is_rejected = false
		  AND created_at >= ? AND created_at <= ?
		  AND problem_words IS NOT NULL AND problem_words != '' AND problem_words != 'null'`,
		userID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, WrapDBError(err, "get problem words list")
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.PW != "" {
			out = append(out, row.PW)
		}
	}
	return out, nil
}

// GetSceneStats 一次通过率与完成场景数
func (r *learningReportRepository) GetSceneStats(ctx context.Context, userID string, start, end time.Time) (int, int, error) {
	type cntRow struct {
		Pass1 int `gorm:"column:pass1"`
		Total int `gorm:"column:total"`
	}
	var cr cntRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN match_result IN ('rule_pass','llm_pass') AND attempt = 1 THEN 1 ELSE 0 END), 0) AS pass1,
			COUNT(*) AS total
		FROM scene_messages
		WHERE user_id = ? AND created_at >= ? AND created_at <= ?`,
		userID, start, end).Scan(&cr).Error
	if err != nil {
		return 0, 0, WrapDBError(err, "scene pass stats")
	}
	passRate := 0
	if cr.Total > 0 {
		passRate = int(float64(cr.Pass1) * 100.0 / float64(cr.Total))
		if passRate > 100 {
			passRate = 100
		}
	}

	var completed int64
	err = r.db.WithContext(ctx).Model(&model.SceneSession{}).
		Where("user_id = ? AND status = ? AND created_at >= ? AND created_at <= ?",
			userID, "completed", start, end).
		Count(&completed).Error
	if err != nil {
		return 0, 0, WrapDBError(err, "count completed scenes")
	}
	return passRate, int(completed), nil
}
