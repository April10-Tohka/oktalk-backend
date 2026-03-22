package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// PronunciationSessionRepository 发音练习 v2 会话仓储
type PronunciationSessionRepository interface {
	Create(ctx context.Context, session *model.PronunciationSession) error
	FindByID(ctx context.Context, id string) (*model.PronunciationSession, error)
	FindOngoingByUserAndUnit(ctx context.Context, userID, unitID string) (*model.PronunciationSession, error)
	UpdateCurrentIndex(ctx context.Context, id string, index int) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type pronunciationSessionRepository struct {
	db *gorm.DB
}

// NewPronunciationSessionRepository 构造函数
func NewPronunciationSessionRepository(db *gorm.DB) PronunciationSessionRepository {
	return &pronunciationSessionRepository{db: db}
}

func (r *pronunciationSessionRepository) Create(ctx context.Context, session *model.PronunciationSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *pronunciationSessionRepository) FindByID(ctx context.Context, id string) (*model.PronunciationSession, error) {
	var s model.PronunciationSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pronunciationSessionRepository) FindOngoingByUserAndUnit(ctx context.Context, userID, unitID string) (*model.PronunciationSession, error) {
	var s model.PronunciationSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND unit_id = ? AND status = ?", userID, unitID, "ongoing").
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *pronunciationSessionRepository) UpdateCurrentIndex(ctx context.Context, id string, index int) error {
	return r.db.WithContext(ctx).Model(&model.PronunciationSession{}).
		Where("id = ?", id).
		Update("current_index", index).Error
}

func (r *pronunciationSessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&model.PronunciationSession{}).
		Where("id = ?", id).
		Update("status", status).Error
}
