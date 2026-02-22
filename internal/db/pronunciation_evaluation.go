// Package db 提供发音评测数据库操作
package db

import (
	"context"
	"encoding/json"
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
	GetStatsByUserAndDateRange(ctx context.Context, userID string, start, end time.Time) (*EvaluationStats, error)

	// 更新方法
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateFeedback(ctx context.Context, id string, level, text string, audioURL *string) error
	UpdateScores(ctx context.Context, id string, overall, accuracy, fluency, integrity int) error

	// 预加载方法
	GetWithUser(ctx context.Context, id string) (*model.PronunciationEvaluation, error)

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

// GetWithUser 获取评测记录及其关联用户
func (r *pronunciationEvaluationRepository) GetWithUser(ctx context.Context, id string) (*model.PronunciationEvaluation, error) {
	var evaluation model.PronunciationEvaluation
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&evaluation).Error
	if err != nil {
		return nil, WrapDBError(err, "get pronunciation evaluation with user")
	}
	return &evaluation, nil
}

// EvaluationStats 评测统计结果
type EvaluationStats struct {
	TotalCount        int
	AvgOverallScore   float64
	AvgAccuracyScore  float64
	AvgFluencyScore   float64
	AvgIntegrityScore float64
	SLevelCount       int
	ALevelCount       int
	BLevelCount       int
	CLevelCount       int
	ProblemWords      map[string]int // word -> frequency
}

// GetStatsByUserAndDateRange 获取用户在指定时间范围内的评测统计（报告用）
func (r *pronunciationEvaluationRepository) GetStatsByUserAndDateRange(ctx context.Context, userID string, start, end time.Time) (*EvaluationStats, error) {
	stats := &EvaluationStats{ProblemWords: make(map[string]int)}

	// 1. 聚合查询：总数、平均分、级别分布
	var result struct {
		TotalCount        int     `gorm:"column:total_count"`
		AvgOverallScore   float64 `gorm:"column:avg_overall"`
		AvgAccuracyScore  float64 `gorm:"column:avg_accuracy"`
		AvgFluencyScore   float64 `gorm:"column:avg_fluency"`
		AvgIntegrityScore float64 `gorm:"column:avg_integrity"`
		SLevelCount       int     `gorm:"column:s_count"`
		ALevelCount       int     `gorm:"column:a_count"`
		BLevelCount       int     `gorm:"column:b_count"`
		CLevelCount       int     `gorm:"column:c_count"`
	}
	err := r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND status = ? AND created_at >= ? AND created_at < ?", userID, model.EvaluationStatusCompleted, start, end).
		Select(
			"COUNT(*) as total_count, " +
				"COALESCE(AVG(overall_score), 0) as avg_overall, " +
				"COALESCE(AVG(accuracy_score), 0) as avg_accuracy, " +
				"COALESCE(AVG(fluency_score), 0) as avg_fluency, " +
				"COALESCE(AVG(integrity_score), 0) as avg_integrity, " +
				"SUM(CASE WHEN feedback_level = 'S' THEN 1 ELSE 0 END) as s_count, " +
				"SUM(CASE WHEN feedback_level = 'A' THEN 1 ELSE 0 END) as a_count, " +
				"SUM(CASE WHEN feedback_level = 'B' THEN 1 ELSE 0 END) as b_count, " +
				"SUM(CASE WHEN feedback_level = 'C' THEN 1 ELSE 0 END) as c_count",
		).
		Scan(&result).Error
	if err != nil {
		return nil, WrapDBError(err, "get evaluation stats by date range")
	}

	stats.TotalCount = result.TotalCount
	stats.AvgOverallScore = result.AvgOverallScore
	stats.AvgAccuracyScore = result.AvgAccuracyScore
	stats.AvgFluencyScore = result.AvgFluencyScore
	stats.AvgIntegrityScore = result.AvgIntegrityScore
	stats.SLevelCount = result.SLevelCount
	stats.ALevelCount = result.ALevelCount
	stats.BLevelCount = result.BLevelCount
	stats.CLevelCount = result.CLevelCount

	// 2. 聚合问题单词（从 problem_words JSON 数组中提取）
	var problemWordsRows []struct {
		ProblemWords string `gorm:"column:problem_words"`
	}
	err = r.db.WithContext(ctx).
		Model(&model.PronunciationEvaluation{}).
		Where("user_id = ? AND status = ? AND created_at >= ? AND created_at < ? AND problem_words IS NOT NULL AND problem_words != '[]' AND problem_words != 'null'",
			userID, model.EvaluationStatusCompleted, start, end).
		Select("problem_words").
		Scan(&problemWordsRows).Error
	if err == nil {
		for _, row := range problemWordsRows {
			var words []string
			if jsonErr := json.Unmarshal([]byte(row.ProblemWords), &words); jsonErr == nil {
				for _, w := range words {
					stats.ProblemWords[w]++
				}
			}
		}
	}

	return stats, nil
}
