package domain

import (
	"context"
	"fmt"
)

// Limiter 限流器核心接口
type Limiter interface {
	// Allow 检查是否允许通过
	// key: 限流维度标识
	//   用户级: "user:{user_id}:{scene}"
	//   全局级: "global:{scene}"
	// 返回 true 表示允许，false 表示被限流
	Allow(ctx context.Context, key string) (bool, error)
}

// LayeredLimiter 分层限流器接口
type LayeredLimiter interface {
	// Check 依次检查用户级和全局级，任一触发返回限流结果
	Check(ctx context.Context, userID string) (*CheckResult, error)
}

// CheckResult 限流检查结果
type CheckResult struct {
	Allowed       bool   // 是否允许
	LimitedBy     string // 触发限流的层："user" / "global" / ""
	RetryAfterSec int64  // 建议等待秒数
	Scene         string // 场景标识
}

// SceneLimiterFactory 场景限流器工厂
type SceneLimiterFactory interface {
	GetLimiter(scene string) (LayeredLimiter, error)
}

var ErrSceneNotFound = fmt.Errorf("rate limit scene not found")

// RateLimitError 限流错误，携带重试信息，供 Handler 层识别并返回 429
type RateLimitError struct {
	Scene         string
	LimitedBy     string
	RetryAfterSec int64
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: scene=%s, limited_by=%s, retry_after=%ds",
		e.Scene, e.LimitedBy, e.RetryAfterSec)
}
