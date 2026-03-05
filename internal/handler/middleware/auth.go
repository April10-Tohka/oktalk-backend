package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/config"
	pkgjwt "pronunciation-correction-system/internal/pkg/jwt"
)

// 定义自定义 key 类型，避免与其他库的 key 冲突
type contextKey string

const (
	UserIDKey contextKey = "user_id"
	JTIKey    contextKey = "jti"
	StatusKey contextKey = "status"
)

// Auth JWT 认证中间件
// cfg: 应用配置，rdb: Redis 客户端（用于黑名单检查）
func Auth(cfg *config.Config, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// dev 模式：直接放行并写入固定 user_id
		if cfg != nil && strings.EqualFold(cfg.Server.Environment, "development") {
			c.Set(string(UserIDKey), "dev-user-123")
			c.Set(string(JTIKey), "dev-jti-123")
			c.Set(string(StatusKey), "active")
			ctx := context.WithValue(c.Request.Context(), UserIDKey, "dev-user-123")
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		// 1. 从 Header 取 Authorization，提取 Bearer token
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "message": "请重新登录", "data": nil,
			})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "message": "请重新登录", "data": nil,
			})
			return
		}

		// 2. 解析 Access Token
		claims, err := pkgjwt.ParseAccessToken(tokenString)
		if err != nil {
			slog.Warn("JWT parse failed", slog.String("error", err.Error()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "message": "请重新登录", "data": nil,
			})
			return
		}

		// 3. 检查黑名单
		if rdb != nil {
			blacklistKey := fmt.Sprintf(cache.KeyTokenBlacklist, claims.JTI)
			exists, err := rdb.Exists(c.Request.Context(), blacklistKey).Result()
			if err != nil {
				slog.Error("Redis blacklist check failed", slog.String("error", err.Error()))
				// Redis 不可用时不阻断请求，继续放行
			} else if exists > 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code": 401, "message": "登录状态已失效", "data": nil,
				})
				return
			}
		}

		// 4. 将 user_id、jti、status 注入 gin.Context
		c.Set(string(UserIDKey), claims.UserID)
		c.Set(string(JTIKey), claims.JTI)
		c.Set(string(StatusKey), claims.Status)

		ctx := context.WithValue(c.Request.Context(), UserIDKey, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		// 5. c.Next()
		c.Next()
	}
}
