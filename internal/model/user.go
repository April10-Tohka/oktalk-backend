// Package model 定义用户相关数据模型
package model

import (
	"time"
)

// User 用户基础信息表
// 存储用户的基础账户信息
type User struct {
	// ID 用户唯一标识 (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// Username 用户名，唯一
	Username string `gorm:"uniqueIndex;type:varchar(100);not null" json:"username" validate:"required,min=3,max=100"`
	// Email 邮箱地址，唯一
	Email string `gorm:"uniqueIndex;type:varchar(255);not null" json:"email" validate:"required,email,max=255"`
	// PasswordHash 密码哈希值，不序列化到 JSON
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-" validate:"required"`
	// Phone 手机号，可选
	Phone *string `gorm:"type:varchar(20)" json:"phone,omitempty" validate:"omitempty,max=20"`
	// AvatarURL 头像 URL
	AvatarURL *string `gorm:"type:varchar(500)" json:"avatar_url,omitempty" validate:"omitempty,url,max=500"`
	// Grade 年级 (1-6 代表小学 1-6 年级)
	Grade *int `gorm:"type:int" json:"grade,omitempty" validate:"omitempty,min=1,max=6"`
	// Status 账户状态：active/suspended/deleted
	Status string `gorm:"type:enum('active','suspended','deleted');default:'active';not null" json:"status" validate:"required,oneof=active suspended deleted"`
	// Language 语言偏好：en/zh
	Language string `gorm:"type:varchar(10);default:'en';not null" json:"language" validate:"required,oneof=en zh"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;index" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`
	// DeletedAt 软删除时间
	DeletedAt *time.Time `gorm:"type:timestamp;index" json:"deleted_at,omitempty"`

	// 关联关系
	Profile            *UserProfile              `gorm:"foreignKey:UserID" json:"profile,omitempty"`
	VoiceConversations []*VoiceConversation      `gorm:"foreignKey:UserID" json:"conversations,omitempty"`
	Evaluations        []*PronunciationEvaluation `gorm:"foreignKey:UserID" json:"evaluations,omitempty"`
	Reports            []*LearningReport         `gorm:"foreignKey:UserID" json:"reports,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserProfile 用户扩展信息表
// 存储用户的详细个人信息和学习进度
type UserProfile struct {
	// ID 主键 (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// UserID 用户 ID，外键
	UserID string `gorm:"uniqueIndex;type:varchar(36);not null" json:"user_id" validate:"required,uuid"`
	// FullName 真实姓名
	FullName *string `gorm:"type:varchar(100)" json:"full_name,omitempty" validate:"omitempty,max=100"`
	// Age 年龄
	Age *int `gorm:"type:int" json:"age,omitempty" validate:"omitempty,min=1,max=150"`
	// Gender 性别：male/female/other
	Gender *string `gorm:"type:enum('male','female','other')" json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
	// Bio 个人简介
	Bio *string `gorm:"type:text" json:"bio,omitempty" validate:"omitempty,max=1000"`
	// TotalConversations 总对话数
	TotalConversations int `gorm:"type:int;default:0;not null" json:"total_conversations" validate:"gte=0"`
	// TotalEvaluations 总评测数
	TotalEvaluations int `gorm:"type:int;default:0;not null" json:"total_evaluations" validate:"gte=0"`
	// TotalReports 总报告数
	TotalReports int `gorm:"type:int;default:0;not null" json:"total_reports" validate:"gte=0"`
	// AverageEvaluationScore 平均评测分数
	AverageEvaluationScore float64 `gorm:"type:float;default:0.0;not null" json:"average_evaluation_score" validate:"gte=0,lte=100"`
	// LastConversationAt 上次对话时间
	LastConversationAt *time.Time `gorm:"type:timestamp;index" json:"last_conversation_at,omitempty"`
	// LastEvaluationAt 上次评测时间
	LastEvaluationAt *time.Time `gorm:"type:timestamp;index" json:"last_evaluation_at,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}
