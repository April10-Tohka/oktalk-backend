package repository

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// PronunciationRecordRepository 发音练习 v2 记录仓储
type PronunciationRecordRepository interface {
	Create(ctx context.Context, record *model.PronunciationRecord) error
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.PronunciationRecord, error)
	GetBestScorePerItem(ctx context.Context, sessionID string) (map[int]float32, error)
}

type pronunciationRecordRepository struct {
	db *gorm.DB
}

// NewPronunciationRecordRepository 构造函数
func NewPronunciationRecordRepository(db *gorm.DB) PronunciationRecordRepository {
	return &pronunciationRecordRepository{db: db}
}

func (r *pronunciationRecordRepository) Create(ctx context.Context, record *model.PronunciationRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *pronunciationRecordRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.PronunciationRecord, error) {
	var list []*model.PronunciationRecord
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *pronunciationRecordRepository) GetBestScorePerItem(ctx context.Context, sessionID string) (map[int]float32, error) {
	type row struct {
		ItemID   int     `gorm:"column:item_id"`
		MaxScore float32 `gorm:"column:max_score"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT item_id, MAX(raw_score) AS max_score
		FROM pronunciation_records
		WHERE session_id = ?
		GROUP BY item_id`, sessionID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]float32, len(rows))
	for _, r := range rows {
		out[r.ItemID] = r.MaxScore
	}
	return out, nil
}
