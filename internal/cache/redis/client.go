// Package redis 提供 Redis 客户端初始化和连接池管理
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"pronunciation-correction-system/internal/config"
)

// Client Redis 客户端封装
type Client struct {
	rdb *redis.Client
}

// ClientConfig Redis 客户端配置
type ClientConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接
	MaxRetries   int           // 命令重试次数
	DialTimeout  time.Duration // 连接超时
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时
}

// DefaultClientConfig 默认配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     50, // 连接池大小
		MinIdleConns: 10, // 最小空闲连接
		MaxRetries:   3,  // 命令重试次数
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewClient 创建 Redis 客户端（使用配置结构体）
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	// 创建客户端配置
	clientCfg := &ClientConfig{
		Host:         cfg.Host,
		Port:         cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	return NewClientWithConfig(clientCfg)
}

// NewClientWithConfig 使用完整配置创建 Redis 客户端
func NewClientWithConfig(cfg *ClientConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// GetClient 获取原始 Redis 客户端
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// PoolStats 获取连接池统计
func (c *Client) PoolStats() *redis.PoolStats {
	return c.rdb.PoolStats()
}

// Pipeline 创建管道
func (c *Client) Pipeline() redis.Pipeliner {
	return c.rdb.Pipeline()
}

// TxPipeline 创建事务管道
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.rdb.TxPipeline()
}
