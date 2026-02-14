// Package http 提供 HTTP 响应工具
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/pkg/errors"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	statusCode := errors.HTTPStatusCode(err)
	response := errors.ToResponse(err)
	c.JSON(statusCode, response)
}

// ErrorWithCode 指定错误码的响应
func ErrorWithCode(c *gin.Context, code int, message string) {
	statusCode := http.StatusBadRequest
	if code >= 5000 {
		statusCode = http.StatusInternalServerError
	}
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 400 错误响应
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    errors.CodeInvalidParam,
		Message: message,
	})
}

// Unauthorized 401 错误响应
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    errors.CodeUnauthorized,
		Message: message,
	})
}

// Forbidden 403 错误响应
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    errors.CodeForbidden,
		Message: message,
	})
}

// NotFound 404 错误响应
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    errors.CodeNotFound,
		Message: message,
	})
}

// InternalError 500 错误响应
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    errors.CodeInternalError,
		Message: message,
	})
}

// Paginated 分页响应
type PaginatedResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// SuccessWithPagination 带分页的成功响应
func SuccessWithPagination(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
