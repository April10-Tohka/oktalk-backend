package xf

import (
	"fmt"
)

// Error 讯飞 API 错误
type Error struct {
	Code    int
	Message string
}

// Error 实现 error 接口
func (e *Error) Error() string {
	return fmt.Sprintf("xunfei api error: code=%d, message=%s", e.Code, e.Message)
}

// NewError 创建错误
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// 常见错误码
const (
	ErrCodeSuccess       = 0     // 成功
	ErrCodeInvalidParam  = 10001 // 参数错误
	ErrCodeAuthFailed    = 10002 // 认证失败
	ErrCodeRateLimited   = 10003 // 请求过于频繁
	ErrCodeQuotaExceeded = 10004 // 配额不足
	ErrCodeInternalError = 10005 // 内部错误
	ErrCodeAudioTooLong  = 10006 // 音频过长
	ErrCodeAudioInvalid  = 10007 // 音频无效
	ErrCodeTextEmpty     = 10008 // 文本为空
	ErrCodeTextTooLong   = 10009 // 文本过长
	ErrCodeServiceBusy   = 10010 // 服务繁忙
)

// 常见错误
var (
	ErrInvalidParam  = NewError(ErrCodeInvalidParam, "invalid parameter")
	ErrAuthFailed    = NewError(ErrCodeAuthFailed, "authentication failed")
	ErrRateLimited   = NewError(ErrCodeRateLimited, "rate limited")
	ErrQuotaExceeded = NewError(ErrCodeQuotaExceeded, "quota exceeded")
	ErrInternalError = NewError(ErrCodeInternalError, "internal error")
	ErrAudioTooLong  = NewError(ErrCodeAudioTooLong, "audio too long")
	ErrAudioInvalid  = NewError(ErrCodeAudioInvalid, "audio invalid")
	ErrTextEmpty     = NewError(ErrCodeTextEmpty, "text is empty")
	ErrTextTooLong   = NewError(ErrCodeTextTooLong, "text too long")
	ErrServiceBusy   = NewError(ErrCodeServiceBusy, "service busy")
)

// ParseError 解析错误码
func ParseError(code int) *Error {
	switch code {
	case ErrCodeSuccess:
		return nil
	case ErrCodeInvalidParam:
		return ErrInvalidParam
	case ErrCodeAuthFailed:
		return ErrAuthFailed
	case ErrCodeRateLimited:
		return ErrRateLimited
	case ErrCodeQuotaExceeded:
		return ErrQuotaExceeded
	case ErrCodeAudioTooLong:
		return ErrAudioTooLong
	case ErrCodeAudioInvalid:
		return ErrAudioInvalid
	case ErrCodeTextEmpty:
		return ErrTextEmpty
	case ErrCodeTextTooLong:
		return ErrTextTooLong
	case ErrCodeServiceBusy:
		return ErrServiceBusy
	default:
		return NewError(code, "unknown error")
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if xfErr, ok := err.(*Error); ok {
		switch xfErr.Code {
		case ErrCodeRateLimited, ErrCodeServiceBusy, ErrCodeInternalError:
			return true
		}
	}
	return false
}
