// Package errors 定义错误码
package errors

// 错误码定义
const (
	// 通用错误码 (1000-1999)
	CodeSuccess       = 0
	CodeInternalError = 1000
	CodeInvalidParam  = 1001
	CodeUnauthorized  = 1002
	CodeForbidden     = 1003
	CodeNotFound      = 1004
	CodeConflict      = 1005
	CodeTooManyRequests = 1006

	// 用户相关错误码 (2000-2999)
	CodeUserNotFound      = 2000
	CodeUserAlreadyExists = 2001
	CodeInvalidPassword   = 2002
	CodeInvalidToken      = 2003
	CodeTokenExpired      = 2004

	// 评测相关错误码 (3000-3999)
	CodeEvaluationNotFound  = 3000
	CodeEvaluationFailed    = 3001
	CodeAudioInvalid        = 3002
	CodeAudioTooLong        = 3003
	CodeTextEmpty           = 3004
	CodeTextTooLong         = 3005

	// 反馈相关错误码 (4000-4999)
	CodeFeedbackNotFound    = 4000
	CodeFeedbackGenFailed   = 4001

	// 第三方服务错误码 (5000-5999)
	CodeXunFeiError      = 5000
	CodeAliyunOSSError   = 5001
	CodeAliyunTTSError   = 5002
	CodeQwenError        = 5003

	// 缓存相关错误码 (6000-6999)
	CodeCacheError = 6000
	CodeCacheMiss  = 6001

	// 队列相关错误码 (7000-7999)
	CodeQueueError    = 7000
	CodeTaskNotFound  = 7001
	CodeTaskFailed    = 7002
)

// 预定义错误
var (
	ErrInternalError = New(CodeInternalError, "internal server error")
	ErrInvalidParam  = New(CodeInvalidParam, "invalid parameter")
	ErrUnauthorized  = New(CodeUnauthorized, "unauthorized")
	ErrForbidden     = New(CodeForbidden, "forbidden")
	ErrNotFound      = New(CodeNotFound, "resource not found")
	ErrConflict      = New(CodeConflict, "resource conflict")
	ErrTooManyRequests = New(CodeTooManyRequests, "too many requests")

	ErrUserNotFound      = New(CodeUserNotFound, "user not found")
	ErrUserAlreadyExists = New(CodeUserAlreadyExists, "user already exists")
	ErrInvalidPassword   = New(CodeInvalidPassword, "invalid password")
	ErrInvalidToken      = New(CodeInvalidToken, "invalid token")
	ErrTokenExpired      = New(CodeTokenExpired, "token expired")

	ErrEvaluationNotFound = New(CodeEvaluationNotFound, "evaluation not found")
	ErrEvaluationFailed   = New(CodeEvaluationFailed, "evaluation failed")
	ErrAudioInvalid       = New(CodeAudioInvalid, "invalid audio")
	ErrAudioTooLong       = New(CodeAudioTooLong, "audio too long")
	ErrTextEmpty          = New(CodeTextEmpty, "text is empty")
	ErrTextTooLong        = New(CodeTextTooLong, "text too long")

	ErrFeedbackNotFound  = New(CodeFeedbackNotFound, "feedback not found")
	ErrFeedbackGenFailed = New(CodeFeedbackGenFailed, "feedback generation failed")
)
