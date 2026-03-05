// Package ratelimit 提供从配置构建场景限流器的工厂
package ratelimit

import (
	"context"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// sceneLimiterMap 实现 domain.SceneLimiterFactory
type sceneLimiterMap struct {
	limiters map[string]domain.LayeredLimiter
}

// NewSceneLimiterFactory 从配置构建工厂
// 若 cfg.Enabled == false，返回 noopFactory{}
func NewSceneLimiterFactory(cfg config.RateLimitConfig) domain.SceneLimiterFactory {
	if !cfg.Enabled {
		return noopFactory{}
	}

	limiters := make(map[string]domain.LayeredLimiter, len(cfg.Scenes))
	for sceneName, sceneCfg := range cfg.Scenes {
		userInterval := time.Duration(sceneCfg.User.RefillIntervalMs) * time.Millisecond
		globalInterval := time.Duration(sceneCfg.Global.RefillIntervalMs) * time.Millisecond

		userLimiter := NewMemoryLimiter(
			sceneCfg.User.Capacity,
			sceneCfg.User.RefillRate,
			userInterval,
		)
		globalLimiter := NewMemoryLimiter(
			sceneCfg.Global.Capacity,
			sceneCfg.Global.RefillRate,
			globalInterval,
		)

		limiters[sceneName] = NewLayeredLimiter(
			sceneName,
			userLimiter, userInterval, sceneCfg.User.RefillRate,
			globalLimiter, globalInterval, sceneCfg.Global.RefillRate,
		)
	}
	return &sceneLimiterMap{limiters: limiters}
}

// GetLimiter 根据场景名称返回对应的 LayeredLimiter
func (m *sceneLimiterMap) GetLimiter(scene string) (domain.LayeredLimiter, error) {
	l, ok := m.limiters[scene]
	if !ok {
		return nil, domain.ErrSceneNotFound
	}
	return l, nil
}

// noopFactory 全局限流关闭时使用，所有 Check 直接返回 Allowed=true
type noopFactory struct{}

func (n noopFactory) GetLimiter(_ string) (domain.LayeredLimiter, error) {
	return noopLimiter{}, nil
}

// noopLimiter 不做任何限制，始终放行
type noopLimiter struct{}

func (n noopLimiter) Check(_ context.Context, _ string) (*domain.CheckResult, error) {
	return &domain.CheckResult{Allowed: true}, nil
}
