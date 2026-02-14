// Package cache 提供分布式锁实现
// 用于防止并发操作冲突
package cache

import (
	"context"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// DistributedLock 分布式锁
type DistributedLock struct {
	commands *redis.Commands
}

// NewDistributedLock 创建分布式锁
func NewDistributedLock(commands *redis.Commands) *DistributedLock {
	return &DistributedLock{
		commands: commands,
	}
}

// LockInfo 锁信息
type LockInfo struct {
	Key       string        // 锁的 key
	Value     string        // 锁的值（用于释放时验证）
	TTL       time.Duration // 锁的过期时间
	AcquiredAt time.Time    // 获取锁的时间
}

// 默认锁配置
const (
	DefaultLockTTL       = 30 * time.Second  // 默认锁超时时间
	DefaultLockRetry     = 3                 // 默认重试次数
	DefaultLockRetryWait = 100 * time.Millisecond // 默认重试等待时间
)

// TryLock 尝试获取锁（不重试）
func (l *DistributedLock) TryLock(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = DefaultLockTTL
	}
	return l.commands.Lock(ctx, key, value, ttl)
}

// Lock 获取锁（带重试）
func (l *DistributedLock) Lock(ctx context.Context, key, value string, ttl time.Duration, maxRetries int, retryWait time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = DefaultLockTTL
	}
	if maxRetries <= 0 {
		maxRetries = DefaultLockRetry
	}
	if retryWait <= 0 {
		retryWait = DefaultLockRetryWait
	}

	for i := 0; i <= maxRetries; i++ {
		acquired, err := l.commands.Lock(ctx, key, value, ttl)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		// 如果还有重试机会，等待后重试
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(retryWait):
				// 继续重试
			}
		}
	}

	return false, nil
}

// Unlock 释放锁
func (l *DistributedLock) Unlock(ctx context.Context, key, value string) error {
	return l.commands.Unlock(ctx, key, value)
}

// Extend 延长锁的过期时间
func (l *DistributedLock) Extend(ctx context.Context, key string, ttl time.Duration) error {
	return l.commands.Expire(ctx, key, ttl)
}

// IsLocked 检查是否被锁定
func (l *DistributedLock) IsLocked(ctx context.Context, key string) (bool, error) {
	return l.commands.Exists(ctx, key)
}

// GetLockTTL 获取锁的剩余时间
func (l *DistributedLock) GetLockTTL(ctx context.Context, key string) (time.Duration, error) {
	return l.commands.TTL(ctx, key)
}

// ==================== 业务锁 Key 构建器 ====================

// EvaluationLockKey 评测锁 Key
func EvaluationLockKey(evaluationID string) string {
	return redis.Keys.Lock.Evaluation(evaluationID)
}

// UserLockKey 用户锁 Key
func UserLockKey(userID string) string {
	return redis.Keys.Lock.User(userID)
}

// ==================== 便捷方法 ====================

// LockEvaluation 锁定评测
func (l *DistributedLock) LockEvaluation(ctx context.Context, evaluationID, value string) (bool, error) {
	key := EvaluationLockKey(evaluationID)
	return l.Lock(ctx, key, value, DefaultLockTTL, DefaultLockRetry, DefaultLockRetryWait)
}

// UnlockEvaluation 解锁评测
func (l *DistributedLock) UnlockEvaluation(ctx context.Context, evaluationID, value string) error {
	key := EvaluationLockKey(evaluationID)
	return l.Unlock(ctx, key, value)
}

// LockUser 锁定用户
func (l *DistributedLock) LockUser(ctx context.Context, userID, value string) (bool, error) {
	key := UserLockKey(userID)
	return l.Lock(ctx, key, value, DefaultLockTTL, DefaultLockRetry, DefaultLockRetryWait)
}

// UnlockUser 解锁用户
func (l *DistributedLock) UnlockUser(ctx context.Context, userID, value string) error {
	key := UserLockKey(userID)
	return l.Unlock(ctx, key, value)
}

// WithLock 使用锁执行操作
// 自动获取锁、执行操作、释放锁
func (l *DistributedLock) WithLock(ctx context.Context, key, value string, ttl time.Duration, fn func() error) error {
	// 获取锁
	acquired, err := l.Lock(ctx, key, value, ttl, DefaultLockRetry, DefaultLockRetryWait)
	if err != nil {
		return err
	}
	if !acquired {
		return ErrLockNotAcquired
	}

	// 确保释放锁
	defer func() {
		_ = l.Unlock(ctx, key, value)
	}()

	// 执行操作
	return fn()
}

// WithEvaluationLock 使用评测锁执行操作
func (l *DistributedLock) WithEvaluationLock(ctx context.Context, evaluationID, value string, fn func() error) error {
	key := EvaluationLockKey(evaluationID)
	return l.WithLock(ctx, key, value, DefaultLockTTL, fn)
}

// WithUserLock 使用用户锁执行操作
func (l *DistributedLock) WithUserLock(ctx context.Context, userID, value string, fn func() error) error {
	key := UserLockKey(userID)
	return l.WithLock(ctx, key, value, DefaultLockTTL, fn)
}

// 错误定义
var (
	ErrLockNotAcquired = &LockError{Message: "failed to acquire lock"}
)

// LockError 锁错误
type LockError struct {
	Message string
}

func (e *LockError) Error() string {
	return e.Message
}
