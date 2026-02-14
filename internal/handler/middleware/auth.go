// Package middleware 提供 HTTP 中间件
// 包含认证、日志、错误处理等中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// 验证 Bearer token 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// TODO: 验证 JWT token
		// TODO: 解析用户信息并存入上下文
		_ = token

		// 设置用户 ID 到上下文（示例）
		c.Set("user_id", "")
		c.Set("username", "")

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件
// 如果提供了 token 则验证，否则继续
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// 如果提供了 token，则进行验证
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			token := parts[1]
			// TODO: 验证 JWT token
			_ = token
		}

		c.Next()
	}
}
