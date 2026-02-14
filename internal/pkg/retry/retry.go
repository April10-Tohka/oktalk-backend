// Package retry 提供重试机制
package retry

import (
	"context"
	"math"
	"time"
)

// Config 重试配置
type Config struct {
	MaxRetries     int           // 最大重试次数
	InitialBackoff time.Duration // 初始退避时间
	MaxBackoff     time.Duration // 最大退避时间
	Multiplier     float64       // 退避时间乘数
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
	}
}

// RetryFunc 重试函数类型
type RetryFunc func() error

// RetryFuncWithContext 带上下文的重试函数
type RetryFuncWithContext func(ctx context.Context) error

// Retry 重试执行
func Retry(fn RetryFunc, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var lastErr error
	backoff := cfg.InitialBackoff

	for i := 0; i <= cfg.MaxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < cfg.MaxRetries {
			time.Sleep(backoff)
			backoff = time.Duration(math.Min(
				float64(backoff)*cfg.Multiplier,
				float64(cfg.MaxBackoff),
			))
		}
	}

	return lastErr
}

// RetryWithContext 带上下文的重试执行
func RetryWithContext(ctx context.Context, fn RetryFuncWithContext, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var lastErr error
	backoff := cfg.InitialBackoff

	for i := 0; i <= cfg.MaxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < cfg.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(
				float64(backoff)*cfg.Multiplier,
				float64(cfg.MaxBackoff),
			))
		}
	}

	return lastErr
}

// Retryer 重试器
type Retryer struct {
	config *Config
}

// NewRetryer 创建重试器
func NewRetryer(cfg *Config) *Retryer {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Retryer{config: cfg}
}

// Do 执行重试
func (r *Retryer) Do(fn RetryFunc) error {
	return Retry(fn, r.config)
}

// DoWithContext 带上下文执行重试
func (r *Retryer) DoWithContext(ctx context.Context, fn RetryFuncWithContext) error {
	return RetryWithContext(ctx, fn, r.config)
}
