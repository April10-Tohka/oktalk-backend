// Package errors 提供错误处理工具
package errors

import (
	"net/http"
)

// HTTPStatusCode 获取 HTTP 状态码
func HTTPStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		switch {
		case appErr.Code == CodeSuccess:
			return http.StatusOK
		case appErr.Code == CodeInvalidParam:
			return http.StatusBadRequest
		case appErr.Code == CodeUnauthorized, appErr.Code == CodeInvalidToken, appErr.Code == CodeTokenExpired:
			return http.StatusUnauthorized
		case appErr.Code == CodeForbidden:
			return http.StatusForbidden
		case appErr.Code == CodeNotFound, appErr.Code == CodeUserNotFound, appErr.Code == CodeEvaluationNotFound, appErr.Code == CodeFeedbackNotFound:
			return http.StatusNotFound
		case appErr.Code == CodeConflict, appErr.Code == CodeUserAlreadyExists:
			return http.StatusConflict
		case appErr.Code == CodeTooManyRequests:
			return http.StatusTooManyRequests
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// ToResponse 转换为响应格式
func ToResponse(err error) map[string]interface{} {
	if appErr, ok := err.(*AppError); ok {
		return map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		}
	}
	return map[string]interface{}{
		"code":    CodeInternalError,
		"message": err.Error(),
	}
}

// Handler 错误处理器
type Handler struct{}

// NewHandler 创建错误处理器
func NewHandler() *Handler {
	return &Handler{}
}

// Handle 处理错误
func (h *Handler) Handle(err error) (int, map[string]interface{}) {
	statusCode := HTTPStatusCode(err)
	response := ToResponse(err)
	return statusCode, response
}

// IsNotFound 判断是否为未找到错误
func IsNotFound(err error) bool {
	return Is(err, ErrNotFound) || 
		Is(err, ErrUserNotFound) || 
		Is(err, ErrEvaluationNotFound) || 
		Is(err, ErrFeedbackNotFound)
}

// IsUnauthorized 判断是否为未授权错误
func IsUnauthorized(err error) bool {
	return Is(err, ErrUnauthorized) || 
		Is(err, ErrInvalidToken) || 
		Is(err, ErrTokenExpired)
}

// IsValidationError 判断是否为参数验证错误
func IsValidationError(err error) bool {
	return Is(err, ErrInvalidParam)
}
