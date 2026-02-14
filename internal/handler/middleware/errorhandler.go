// Package middleware 提供错误处理中间件
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware 全局错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// TODO: 根据错误类型返回不同的响应
			// TODO: 记录错误日志

			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": err.Error(),
			})
			return
		}
	}
}

// RecoveryMiddleware panic 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// TODO: 记录 panic 日志
				// TODO: 发送告警通知

				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string, details interface{}) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}
