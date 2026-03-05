package service

import (
	"context"
	"log/slog"

	domain "pronunciation-correction-system/internal/domain"
)

// checkRateLimit 统一限流检查，供三个 Service 复用
func checkRateLimit(
	ctx context.Context,
	factory domain.SceneLimiterFactory,
	scene, userID string,
	logger *slog.Logger,
) error {
	if factory == nil {
		if logger != nil {
			logger.Warn("rate limit factory not initialized, skipping limit check", "scene", scene)
		}
		return nil
	}

	limiter, err := factory.GetLimiter(scene)
	if err != nil {
		// 场景未配置：记录 warn，降级放行
		if logger != nil {
			logger.Warn("rate limit scene not found", "scene", scene)
		}
		return nil
	}

	result, err := limiter.Check(ctx, userID)
	if err != nil {
		// 检查本身出错：记录 error，降级放行
		if logger != nil {
			logger.Error("rate limiter check error", "scene", scene, "error", err)
		}
		return nil
	}

	if !result.Allowed {
		return &domain.RateLimitError{
			Scene:         result.Scene,
			LimitedBy:     result.LimitedBy,
			RetryAfterSec: result.RetryAfterSec,
		}
	}
	return nil
}
