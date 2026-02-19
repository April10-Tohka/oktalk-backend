package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"pronunciation-correction-system/internal/config"
)

// 定义自定义 key 类型，避免与其他库的 key 冲突
type contextKey string

const UserIDKey contextKey = "user_id"

// Auth JWT 认证中间件
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// dev 模式：直接放行并写入固定 user_id
		if cfg != nil && strings.EqualFold(cfg.Server.Environment, "development") {
			// gin.Context 中设置 user_id
			c.Set(string(UserIDKey), "dev-user-123")
			// context.WithValue 中设置 user_id
			ctx := context.WithValue(c.Request.Context(), UserIDKey, "dev-user-123")
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		// prod 模式：校验 Authorization Bearer Token
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized", "data": nil})
			return
		}
		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized", "data": nil})
			return
		}
		if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized", "data": nil})
			return
		}

		claims := jwt.MapClaims{}
		parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized", "data": nil})
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok || strings.TrimSpace(userID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized", "data": nil})
			return
		}
		// 校验通过，将 user_id 写入 gin.Context
		c.Set(string(UserIDKey), userID)
		// context.WithValue 中设置 user_id
		ctx := context.WithValue(c.Request.Context(), UserIDKey, userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
