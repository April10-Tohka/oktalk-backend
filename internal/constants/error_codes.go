// Package constants 定义错误码常量
package constants

// 通用错误码 (1000-1999)
const (
	ErrCodeSuccess            = 0
	ErrCodeInternalError      = 1000
	ErrCodeInvalidParam       = 1001
	ErrCodeUnauthorized       = 1002
	ErrCodeForbidden          = 1003
	ErrCodeNotFound           = 1004
	ErrCodeConflict           = 1005
	ErrCodeTooManyRequests    = 1006
	ErrCodeTimeout            = 1007
	ErrCodeServiceUnavailable = 1008
)

// 用户相关错误码 (2000-2999)
const (
	ErrCodeUserNotFound      = 2000
	ErrCodeUserAlreadyExists = 2001
	ErrCodeInvalidPassword   = 2002
	ErrCodeInvalidToken      = 2003
	ErrCodeTokenExpired      = 2004
	ErrCodeUserDisabled      = 2005
	ErrCodeInvalidEmail      = 2006
	ErrCodeInvalidUsername   = 2007
)

// 评测相关错误码 (3000-3999)
const (
	ErrCodeEvaluationNotFound     = 3000
	ErrCodeEvaluationFailed       = 3001
	ErrCodeAudioInvalid           = 3002
	ErrCodeAudioTooLong           = 3003
	ErrCodeAudioTooShort          = 3004
	ErrCodeAudioFormatUnsupported = 3005
	ErrCodeTextEmpty              = 3006
	ErrCodeTextTooLong            = 3007
	ErrCodeEvaluationInProgress   = 3008
)

// 反馈相关错误码 (4000-4999)
const (
	ErrCodeFeedbackNotFound   = 4000
	ErrCodeFeedbackGenFailed  = 4001
	ErrCodeFeedbackInProgress = 4002
)

// 第三方服务错误码 (5000-5999)
const (
	ErrCodeXunFeiError      = 5000
	ErrCodeXunFeiAuthFailed = 5001
	ErrCodeXunFeiRateLimit  = 5002
	ErrCodeAliyunOSSError   = 5010
	ErrCodeAliyunTTSError   = 5020
	ErrCodeQwenError        = 5030
	ErrCodeQwenRateLimit    = 5031
)

// 缓存相关错误码 (6000-6999)
const (
	ErrCodeCacheError   = 6000
	ErrCodeCacheMiss    = 6001
	ErrCodeCacheExpired = 6002
)

// 队列相关错误码 (7000-7999)
const (
	ErrCodeQueueError   = 7000
	ErrCodeTaskNotFound = 7001
	ErrCodeTaskFailed   = 7002
	ErrCodeQueueFull    = 7003
)

// 对话相关错误码 (8000-8999)
const (
	ErrCodeSessionNotFound = 8000
	ErrCodeSessionClosed   = 8001
	ErrCodeMessageTooLong  = 8002
)

// 报告相关错误码 (9000-9999)
const (
	ErrCodeReportNotFound   = 9000
	ErrCodeReportGenFailed  = 9001
	ErrCodeReportInProgress = 9002
)

// ErrorMessages 错误消息映射
var ErrorMessages = map[int]string{
	ErrCodeSuccess:            "success",
	ErrCodeInternalError:      "internal server error",
	ErrCodeInvalidParam:       "invalid parameter",
	ErrCodeUnauthorized:       "unauthorized",
	ErrCodeForbidden:          "forbidden",
	ErrCodeNotFound:           "resource not found",
	ErrCodeConflict:           "resource conflict",
	ErrCodeTooManyRequests:    "too many requests",
	ErrCodeTimeout:            "request timeout",
	ErrCodeServiceUnavailable: "service unavailable",

	ErrCodeUserNotFound:      "user not found",
	ErrCodeUserAlreadyExists: "user already exists",
	ErrCodeInvalidPassword:   "invalid password",
	ErrCodeInvalidToken:      "invalid token",
	ErrCodeTokenExpired:      "token expired",
	ErrCodeUserDisabled:      "user is disabled",

	ErrCodeEvaluationNotFound: "evaluation not found",
	ErrCodeEvaluationFailed:   "evaluation failed",
	ErrCodeAudioInvalid:       "invalid audio",
	ErrCodeAudioTooLong:       "audio too long",
	ErrCodeTextEmpty:          "text is empty",
	ErrCodeTextTooLong:        "text too long",

	ErrCodeFeedbackNotFound:  "feedback not found",
	ErrCodeFeedbackGenFailed: "feedback generation failed",

	ErrCodeXunFeiError:    "xunfei api error",
	ErrCodeAliyunOSSError: "aliyun oss error",
	ErrCodeAliyunTTSError: "aliyun tts error",
	ErrCodeQwenError:      "qwen api error",
}

// GetErrorMessage 获取错误消息
func GetErrorMessage(code int) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "unknown error"
}
