// Package ratelimit 提供基于内存 map 的限流器
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryLimiter 基于内存 map 的限流器，实现 domain.Limiter
type MemoryLimiter struct {
	buckets        map[string]*tokenBucket
	mu             sync.Mutex
	capacity       int64
	refillRate     int64
	refillInterval time.Duration
}

// NewMemoryLimiter 构造函数
func NewMemoryLimiter(capacity, refillRate int64, refillInterval time.Duration) *MemoryLimiter {
	return &MemoryLimiter{
		buckets:        make(map[string]*tokenBucket),
		capacity:       capacity,
		refillRate:     refillRate,
		refillInterval: refillInterval,
	}
}

// Allow 实现 domain.Limiter 接口
func (m *MemoryLimiter) Allow(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	bucket, ok := m.buckets[key]
	if !ok {
		bucket = newTokenBucket(m.capacity, m.refillRate, m.refillInterval)
		m.buckets[key] = bucket
	}
	m.mu.Unlock()

	return bucket.allow(), nil
}
