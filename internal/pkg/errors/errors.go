// Package errors 提供自定义错误类型
package errors

import (
	"fmt"
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回包装的错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithMessage 添加消息
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: fmt.Sprintf("%s: %s", e.Message, msg),
		Err:     e.Err,
	}
}

// WithError 添加错误
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// Is 判断错误是否相等
func Is(err error, target *AppError) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == target.Code
	}
	return false
}

// GetCode 获取错误码
func GetCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return CodeInternalError
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}
	return err.Error()
}
