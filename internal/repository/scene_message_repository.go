package repository

import (
	"context"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/model"
)

// SceneMessageRepository 场景消息仓储
type SceneMessageRepository interface {
	Create(ctx context.Context, msg *model.SceneMessage) error
	CountAttempts(ctx context.Context, sessionID string, stepID int) (int, error)
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.SceneMessage, error)
	CountPassedSteps(ctx context.Context, sessionID string) (int, error)
}

type sceneMessageRepository struct {
	db *gorm.DB
}

// NewSceneMessageRepository 构造函数
func NewSceneMessageRepository(db *gorm.DB) SceneMessageRepository {
	return &sceneMessageRepository{db: db}
}

func (r *sceneMessageRepository) Create(ctx context.Context, msg *model.SceneMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *sceneMessageRepository) CountAttempts(ctx context.Context, sessionID string, stepID int) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.SceneMessage{}).
		Where("session_id = ? AND step_id = ?", sessionID, stepID).
		Count(&n).Error
	return int(n), err
}

func (r *sceneMessageRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.SceneMessage, error) {
	var list []*model.SceneMessage
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *sceneMessageRepository) CountPassedSteps(ctx context.Context, sessionID string) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT step_id) FROM scene_messages
		WHERE session_id = ? AND match_result IN ('rule_pass','llm_pass')`,
		sessionID).Scan(&n).Error
	return int(n), err
}
