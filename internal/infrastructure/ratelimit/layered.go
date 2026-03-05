// Package ratelimit 提供分层限流器（用户级 + 全局级）
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"pronunciation-correction-system/internal/domain"
)

// layeredLimiter 分层限流器，实现 domain.LayeredLimiter
type layeredLimiter struct {
	scene          string
	userLimiter    domain.Limiter
	globalLimiter  domain.Limiter
	userInterval   time.Duration
	globalInterval time.Duration
	userRate       int64
	globalRate     int64
}

// NewLayeredLimiter 构造函数
func NewLayeredLimiter(
	scene string,
	userLimiter domain.Limiter,
	userInterval time.Duration,
	userRate int64,
	globalLimiter domain.Limiter,
	globalInterval time.Duration,
	globalRate int64,
) domain.LayeredLimiter {
	return &layeredLimiter{
		scene:          scene,
		userLimiter:    userLimiter,
		globalLimiter:  globalLimiter,
		userInterval:   userInterval,
		globalInterval: globalInterval,
		userRate:       userRate,
		globalRate:     globalRate,
	}
}

// Check 分层检查：先用户级 → 再全局级
func (l *layeredLimiter) Check(ctx context.Context, userID string) (*domain.CheckResult, error) {
	// 步骤 1：检查用户级
	userKey := fmt.Sprintf("user:%s:%s", userID, l.scene)
	userAllowed, _ := l.userLimiter.Allow(ctx, userKey)
	if !userAllowed {
		retryAfter := retrySeconds(l.userInterval, l.userRate)
		return &domain.CheckResult{
			Allowed:       false,
			LimitedBy:     "user",
			RetryAfterSec: retryAfter,
			Scene:         l.scene,
		}, nil
	}

	// 步骤 2：检查全局级
	globalKey := fmt.Sprintf("global:%s", l.scene)
	globalAllowed, _ := l.globalLimiter.Allow(ctx, globalKey)
	if !globalAllowed {
		retryAfter := retrySeconds(l.globalInterval, l.globalRate)
		return &domain.CheckResult{
			Allowed:       false,
			LimitedBy:     "global",
			RetryAfterSec: retryAfter,
			Scene:         l.scene,
		}, nil
	}

	// 步骤 3：两层均通过
	return &domain.CheckResult{
		Allowed: true,
		Scene:   l.scene,
	}, nil
}

// retrySeconds 计算建议的等待秒数（最小为 1）
func retrySeconds(interval time.Duration, rate int64) int64 {
	if rate <= 0 {
		return 1
	}
	secs := math.Ceil(float64(interval) / float64(time.Second) / float64(rate))
	if secs < 1 {
		return 1
	}
	return int64(secs)
}
