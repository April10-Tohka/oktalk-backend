package model

import "time"

// SceneSession 场景引导会话
type SceneSession struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"                    json:"id"`
	UserID      string    `gorm:"type:varchar(36);not null;index"                json:"user_id"`
	SceneID     string    `gorm:"type:varchar(50);not null"                      json:"scene_id"`
	CurrentStep int       `gorm:"type:int;not null;default:1"                    json:"current_step"`
	Status      string    `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"                                 json:"updated_at"`
}

// TableName 表名
func (SceneSession) TableName() string { return "scene_sessions" }
