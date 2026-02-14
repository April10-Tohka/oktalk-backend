// Package db 定义数据库操作相关错误
package db

import (
	"errors"

	"gorm.io/gorm"

	apperr "pronunciation-correction-system/internal/pkg/errors"
)

// 数据库错误码 (8000-8999)
const (
	CodeDBError       = 8000 // 数据库通用错误
	CodeDBNotFound    = 8001 // 记录不存在
	CodeDBDuplicate   = 8002 // 记录重复
	CodeDBTransaction = 8003 // 事务错误
	CodeDBConnection  = 8004 // 连接错误
	CodeDBTimeout     = 8005 // 超时错误
	CodeDBInvalidData = 8006 // 数据无效
)

// 预定义数据库错误
var (
	ErrDBError       = apperr.New(CodeDBError, "database error")
	ErrDBNotFound    = apperr.New(CodeDBNotFound, "record not found")
	ErrDBDuplicate   = apperr.New(CodeDBDuplicate, "record already exists")
	ErrDBTransaction = apperr.New(CodeDBTransaction, "transaction error")
	ErrDBConnection  = apperr.New(CodeDBConnection, "database connection error")
	ErrDBTimeout     = apperr.New(CodeDBTimeout, "database timeout")
	ErrDBInvalidData = apperr.New(CodeDBInvalidData, "invalid data")
)

// WrapDBError 包装数据库错误为业务错误
// 根据 GORM 错误类型返回对应的业务错误
func WrapDBError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// 记录不存在
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrDBNotFound.WithMessage(operation)
	}

	// 重复键错误（通常是 MySQL 1062 错误）
	if isDuplicateKeyError(err) {
		return ErrDBDuplicate.WithMessage(operation)
	}

	// 其他错误
	return apperr.Wrap(CodeDBError, operation, err)
}

// isDuplicateKeyError 检查是否为重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// MySQL duplicate entry error
	return contains(errStr, "Duplicate entry") || contains(errStr, "1062")
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsNotFound 检查错误是否为记录不存在
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	if appErr, ok := err.(*apperr.AppError); ok {
		return appErr.Code == CodeDBNotFound
	}
	return false
}

// IsDuplicate 检查错误是否为记录重复
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*apperr.AppError); ok {
		return appErr.Code == CodeDBDuplicate
	}
	return isDuplicateKeyError(err)
}
