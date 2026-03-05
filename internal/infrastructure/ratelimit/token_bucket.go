// Package ratelimit 提供基于内存的令牌桶限流实现
package ratelimit

import (
	"sync"
	"time"
)

// tokenBucket 单个令牌桶，并发安全
type tokenBucket struct {
	capacity       int64         // 桶容量（最大令牌数）
	tokens         int64         // 当前令牌数
	refillRate     int64         // 每个 refillInterval 补充的令牌数
	refillInterval time.Duration // 补充周期
	lastRefill     time.Time     // 上次补充时间戳
	mu             sync.Mutex
}

// newTokenBucket 创建令牌桶，初始令牌数 = capacity（桶满）
func newTokenBucket(capacity, refillRate int64, refillInterval time.Duration) *tokenBucket {
	return &tokenBucket{
		capacity:       capacity,
		tokens:         capacity,
		refillRate:     refillRate,
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// allow 消耗 1 个令牌，返回 true 表示允许；false 表示被限流
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 步骤 1：计算已过了多少个完整补充周期
	elapsed := int64(time.Since(tb.lastRefill) / tb.refillInterval)
	if elapsed > 0 {
		// 步骤 2：补充令牌，不超过桶容量
		added := elapsed * tb.refillRate
		tb.tokens = min64(tb.tokens+added, tb.capacity)
		// 只推进已消耗的完整周期，避免精度丢失
		tb.lastRefill = tb.lastRefill.Add(time.Duration(elapsed) * tb.refillInterval)
	}

	// 步骤 3：消耗令牌
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// remaining 返回当前剩余令牌数（先补充再查询，不消耗令牌）
func (tb *tokenBucket) remaining() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	elapsed := int64(time.Since(tb.lastRefill) / tb.refillInterval)
	if elapsed > 0 {
		added := elapsed * tb.refillRate
		tokens := min64(tb.tokens+added, tb.capacity)
		return tokens
	}
	return tb.tokens
}

// reset 重置令牌数为 capacity，lastRefill 为当前时间
func (tb *tokenBucket) reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// min64 返回两个 int64 中的较小值
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
