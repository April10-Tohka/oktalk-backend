package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Auth JWT 认证中间件（临时放行版本）
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Step2 实现真实 JWT 验证逻辑
		// 1. 从 Header 获取 Authorization: Bearer {token}
		// 2. 验证 token 合法性
		// 3. 解析 user_id 写入 Context: c.Set("user_id", userID)
		// 4. token 非法则 c.AbortWithStatusJSON(401, Response{...})

		// 临时实现：直接放行所有请求，方便 Step1/Step2 开发调试
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("user_id", "dev-user-123")
		} else {
			c.Set("user_id", "dev-user-123")
		}

		c.Next()
	}
}
