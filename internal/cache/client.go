package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"pronunciation-correction-system/internal/config"
)

// ===================== Redis Client 封装 =====================

// NewRedisClient 创建 Redis 客户端，支持连接池配置
func NewRedisClient(cfg config.RedisConfig) *redis.Client {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		// 连接池
		PoolSize:     20,
		MinIdleConns: 5,

		// 超时
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	slog.Info("Redis client created", slog.String("addr", addr), slog.Int("db", cfg.DB))
	return rdb
}

// Ping 健康检查
func Ping(ctx context.Context, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return rdb.Ping(ctx).Err()
}
