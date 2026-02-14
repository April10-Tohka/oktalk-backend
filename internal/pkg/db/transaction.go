// Package db 提供事务管理
package db

import (
	"context"

	"gorm.io/gorm"
)

// TxFunc 事务函数类型
type TxFunc func(tx *gorm.DB) error

// WithTransaction 执行事务
func WithTransaction(ctx context.Context, db *gorm.DB, fn TxFunc) error {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// TransactionManager 事务管理器
type TransactionManager struct {
	db *gorm.DB
}

// NewTransactionManager 创建事务管理器
func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// Execute 执行事务
func (tm *TransactionManager) Execute(ctx context.Context, fn TxFunc) error {
	return WithTransaction(ctx, tm.db, fn)
}

// ExecuteWithResult 执行事务并返回结果
func ExecuteWithResult[T any](ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) (T, error)) (T, error) {
	var result T
	
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return result, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var err error
	result, err = fn(tx)
	if err != nil {
		tx.Rollback()
		return result, err
	}

	if err := tx.Commit().Error; err != nil {
		return result, err
	}

	return result, nil
}
