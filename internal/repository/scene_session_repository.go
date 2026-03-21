package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// SceneSessionRepository 场景会话仓储
type SceneSessionRepository interface {
	Create(ctx context.Context, session *model.SceneSession) error
	FindByID(ctx context.Context, id string) (*model.SceneSession, error)
	FindActiveByUserAndScene(ctx context.Context, userID, sceneID string) (*model.SceneSession, error)
	UpdateCurrentStep(ctx context.Context, id string, step int) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type sceneSessionRepository struct {
	db *gorm.DB
}

// NewSceneSessionRepository 构造函数
func NewSceneSessionRepository(db *gorm.DB) SceneSessionRepository {
	return &sceneSessionRepository{db: db}
}

func (r *sceneSessionRepository) Create(ctx context.Context, session *model.SceneSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sceneSessionRepository) FindByID(ctx context.Context, id string) (*model.SceneSession, error) {
	var s model.SceneSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *sceneSessionRepository) FindActiveByUserAndScene(ctx context.Context, userID, sceneID string) (*model.SceneSession, error) {
	var s model.SceneSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND scene_id = ? AND status = ?", userID, sceneID, "active").
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *sceneSessionRepository) UpdateCurrentStep(ctx context.Context, id string, step int) error {
	return r.db.WithContext(ctx).Model(&model.SceneSession{}).
		Where("id = ?", id).
		Update("current_step", step).Error
}

func (r *sceneSessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&model.SceneSession{}).
		Where("id = ?", id).
		Update("status", status).Error
}
