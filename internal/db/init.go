// Package db 提供数据库操作层
// 负责数据库连接初始化、迁移管理和 Repository 工厂
package db

import (
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/model"
)

// Repositories 包含所有 Repository 实例
type Repositories struct {
	User                    UserRepository
	UserProfile             UserProfileRepository
	VoiceConversation       VoiceConversationRepository
	ConversationMessage     ConversationMessageRepository
	PronunciationEvaluation PronunciationEvaluationRepository
	LearningReport          LearningReportRepository
	SystemSetting           SystemSettingRepository
}

// NewRepositories 创建所有 Repository 实例
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:                    NewUserRepository(db),
		UserProfile:             NewUserProfileRepository(db),
		VoiceConversation:       NewVoiceConversationRepository(db),
		ConversationMessage:     NewConversationMessageRepository(db),
		PronunciationEvaluation: NewPronunciationEvaluationRepository(db),
		LearningReport:          NewLearningReportRepository(db),
		SystemSetting:           NewSystemSettingRepository(db),
	}
}

// WithTx 返回使用事务的 Repositories
func (r *Repositories) WithTx(tx *gorm.DB) *Repositories {
	return &Repositories{
		User:                    r.User.WithTx(tx),
		UserProfile:             r.UserProfile.WithTx(tx),
		VoiceConversation:       r.VoiceConversation.WithTx(tx),
		ConversationMessage:     r.ConversationMessage.WithTx(tx),
		PronunciationEvaluation: r.PronunciationEvaluation.WithTx(tx),
		LearningReport:          r.LearningReport.WithTx(tx),
		SystemSetting:           r.SystemSetting.WithTx(tx),
	}
}

// Init 初始化数据库连接
func Init(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	// 设置日志级别
	logLevel := logger.Silent
	if cfg.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 获取底层 SQL DB 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)

	return db, nil
}

// Migrate 执行数据库迁移
// 自动创建或更新所有表结构（v2.0: 7 张表）
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 用户相关
		&model.User{},
		&model.UserProfile{},
		// 对话相关
		&model.VoiceConversation{},
		&model.ConversationMessage{},
		// 评测相关（已合并 EvaluationDetail 和 FeedbackRecord）
		&model.PronunciationEvaluation{},
		// 报告相关（已合并 ReportStatistic）
		&model.LearningReport{},
		// 系统配置
		&model.SystemSetting{},
	)
}

// InitSystemSettings 初始化系统默认配置
func InitSystemSettings(ctx context.Context, db *gorm.DB) error {
	repo := NewSystemSettingRepository(db)
	return repo.InitDefaults(ctx)
}

// Transaction 执行事务
// fn 接收事务 DB，返回错误时自动回滚，否则自动提交
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// TransactionWithContext 带上下文的事务执行
func TransactionWithContext(ctx context.Context, db *gorm.DB, fn func(ctx context.Context, tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, tx)
	})
}

// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping 检查数据库连接
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
