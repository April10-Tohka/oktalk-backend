// Package model 定义用户相关数据模型
package model

import (
	"time"
)

// User 用户基础信息表
// 存储用户的基础账户信息
// 对应数据库表: users
type User struct {
	// ID 用户唯一标识 (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// Username 用户名，唯一
	Username string `gorm:"uniqueIndex;type:varchar(100);not null" json:"username" validate:"required,min=3,max=100"`
	// Phone 手机号，唯一，可选
	Phone *string `gorm:"uniqueIndex;type:varchar(20)" json:"phone,omitempty" validate:"omitempty,max=20"`
	// RegisterSource  标识该账户是通过哪种方式首次创建的 sms / wechat
	RegisterSource string `gorm:"type:varchar(20);not null"        json:"register_source"`
	// Status 账户状态   active / banned / deactivated
	Status string `gorm:"type:varchar(20);default:'active';not null" json:"status"`
	// AvatarURL 头像 URL
	AvatarURL *string `gorm:"type:varchar(500)" json:"avatar_url,omitempty" validate:"omitempty,url,max=500"`
	// Grade 年级 (1-6 代表小学 1-6 年级)
	Grade *int `gorm:"type:int" json:"grade,omitempty" validate:"omitempty,min=1,max=6"`
	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;index" json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp" json:"updated_at"`
	// DeletedAt 软删除时间
	DeletedAt *time.Time `gorm:"type:timestamp;index" json:"deleted_at,omitempty"`

	// 关联关系
	Profile            *UserProfile               `gorm:"foreignKey:UserID" json:"profile,omitempty"`
	VoiceConversations []*VoiceConversation       `gorm:"foreignKey:UserID" json:"conversations,omitempty"`
	Evaluations        []*PronunciationEvaluation `gorm:"foreignKey:UserID" json:"evaluations,omitempty"`
	Reports            []*LearningReport          `gorm:"foreignKey:UserID" json:"reports,omitempty"`
	WechatBinding      *UserWechatBinding         `gorm:"foreignKey:UserID"             json:"wechat_binding,omitempty"`
	LoginLogs          []*UserLoginLog            `gorm:"foreignKey:UserID"             json:"login_logs,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserProfile 用户扩展信息表
// 存储用户的详细个人信息和学习统计数据
// 对应数据库表: user_profiles
type UserProfile struct {
	// ID 主键 (UUID)
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id" validate:"required,uuid"`
	// UserID 用户 ID，外键，唯一
	UserID string `gorm:"uniqueIndex;type:varchar(36);not null" json:"user_id" validate:"required,uuid"`
	// Age 年龄
	Age *int `gorm:"type:int" json:"age,omitempty" validate:"omitempty,min=1,max=150"`
	// Gender 性别：boy/girl
	Gender *string `gorm:"type:enum('boy','girl')" json:"gender,omitempty" validate:"omitempty,oneof=boy girl"`
	// Bio 个人简介
	Bio *string `gorm:"type:text" json:"bio,omitempty" validate:"omitempty,max=1000"`
	// TotalConversations 总对话数
	TotalConversations int `gorm:"type:int;default:0;not null" json:"total_conversations" validate:"gte=0"`
	// TotalEvaluations 总评测数
	TotalEvaluations int `gorm:"type:int;default:0;not null" json:"total_evaluations" validate:"gte=0"`
	// TotalReports 总报告数
	TotalReports int `gorm:"type:int;default:0;not null" json:"total_reports" validate:"gte=0"`
	// TotalStudyMinutes 累计学习时长（分钟）
	TotalStudyMinutes int `gorm:"type:int;default:0;not null" json:"total_study_minutes" validate:"gte=0"`
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

// UserLoginLog 登录日志
// 用于审计和安全分析，异步写入
// 对应数据库表: user_login_logs
type UserLoginLog struct {
	ID     string `gorm:"primaryKey;type:varchar(36)"        json:"id"`
	UserID string `gorm:"type:varchar(36);not null;index"    json:"user_id"`
	// sms / wechat / token_refresh
	LoginType string `gorm:"type:varchar(20);not null"          json:"login_type"`
	// success / failed
	Result string `gorm:"type:varchar(20);not null"          json:"result"`
	// 失败原因（成功时为空）
	FailReason *string `gorm:"type:varchar(200)"                  json:"fail_reason,omitempty"`
	IP         string  `gorm:"type:varchar(45);not null"          json:"ip"`
	// ios / android / web
	Platform  *string   `gorm:"type:varchar(20)"                   json:"platform,omitempty"`
	DeviceID  *string   `gorm:"type:varchar(100)"                  json:"device_id,omitempty"`
	UserAgent *string   `gorm:"type:varchar(500)"                  json:"user_agent,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;index" json:"created_at"`
}

func (UserLoginLog) TableName() string {
	return "user_login_logs"
}

// UserWechatBinding 微信绑定信息
// 存储用户与微信账号的关联关系
// 对应数据库表: user_wechat_bindings
type UserWechatBinding struct {
	ID     string `gorm:"primaryKey;type:varchar(36)"                    json:"id"`
	UserID string `gorm:"uniqueIndex;type:varchar(36);not null"          json:"user_id"`
	// 微信 App 应用内唯一 ID
	OpenID string `gorm:"uniqueIndex;type:varchar(100);not null"         json:"open_id"`
	// 微信昵称快照（仅首次授权时记录，不实时同步）
	WechatNickname *string `gorm:"type:varchar(100)"                            json:"wechat_nickname,omitempty"`
	// 微信头像快照
	WechatAvatarURL *string `gorm:"type:varchar(500)"                           json:"wechat_avatar_url,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp"                  json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp"                  json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;references:ID"                          json:"user,omitempty"`
}

func (UserWechatBinding) TableName() string {
	return "user_wechat_bindings"
}
