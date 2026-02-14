// Package cache 提供统一的缓存管理器
// 整合所有缓存组件，提供统一的初始化和访问接口
package cache

import (
	"pronunciation-correction-system/internal/cache/redis"
	"pronunciation-correction-system/internal/config"
)

// Manager 缓存管理器
// 整合所有缓存组件
type Manager struct {
	// Redis 客户端
	client   *redis.Client
	commands *redis.Commands

	// 各业务缓存
	Evaluation  *EvaluationCache    // 评测结果缓存
	DemoAudio   *DemoAudioCache     // 示范音频缓存
	Quota       *QuotaCache         // 用户配额缓存
	UploadToken *UploadTokenCache   // 上传令牌缓存
	Feedback    *FeedbackCache      // 反馈缓存
	User        *UserCache          // 用户缓存
	Audio       *AudioCache         // 音频缓存
	Session     *SessionCache       // 会话缓存
	Lock        *DistributedLock    // 分布式锁
	RateLimit   *RateLimitCache     // 限流缓存
}

// ManagerConfig 缓存管理器配置
type ManagerConfig struct {
	Redis        *config.RedisConfig
	DefaultQuota int // 默认每日配额
}

// NewManager 创建缓存管理器
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	// 创建 Redis 客户端
	client, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, err
	}

	// 创建命令封装
	commands := redis.NewCommands(client)

	// 设置默认配额
	defaultQuota := cfg.DefaultQuota
	if defaultQuota <= 0 {
		defaultQuota = 50 // 默认50次/天
	}

	// 创建管理器
	m := &Manager{
		client:   client,
		commands: commands,
	}

	// 初始化各业务缓存
	m.Evaluation = NewEvaluationCache(commands)
	m.DemoAudio = NewDemoAudioCache(commands)
	m.Quota = NewQuotaCache(commands, defaultQuota)
	m.UploadToken = NewUploadTokenCache(commands)
	m.Feedback = NewFeedbackCache(commands)
	m.User = NewUserCache(commands)
	m.Audio = NewAudioCache(commands)
	m.Session = NewSessionCache(commands)
	m.Lock = NewDistributedLock(commands)
	m.RateLimit = NewRateLimitCache(commands)

	return m, nil
}

// NewManagerWithClient 使用现有客户端创建缓存管理器
func NewManagerWithClient(client *redis.Client, defaultQuota int) *Manager {
	commands := redis.NewCommands(client)

	if defaultQuota <= 0 {
		defaultQuota = 50
	}

	m := &Manager{
		client:   client,
		commands: commands,
	}

	m.Evaluation = NewEvaluationCache(commands)
	m.DemoAudio = NewDemoAudioCache(commands)
	m.Quota = NewQuotaCache(commands, defaultQuota)
	m.UploadToken = NewUploadTokenCache(commands)
	m.Feedback = NewFeedbackCache(commands)
	m.User = NewUserCache(commands)
	m.Audio = NewAudioCache(commands)
	m.Session = NewSessionCache(commands)
	m.Lock = NewDistributedLock(commands)
	m.RateLimit = NewRateLimitCache(commands)

	return m
}

// Close 关闭缓存管理器
func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// GetClient 获取 Redis 客户端
func (m *Manager) GetClient() *redis.Client {
	return m.client
}

// GetCommands 获取 Redis 命令封装
func (m *Manager) GetCommands() *redis.Commands {
	return m.commands
}

// PoolStats 获取连接池统计
func (m *Manager) PoolStats() *redis.PoolStatsInfo {
	if m.client == nil {
		return nil
	}
	stats := m.client.PoolStats()
	return &redis.PoolStatsInfo{
		Hits:       stats.Hits,
		Misses:     stats.Misses,
		Timeouts:   stats.Timeouts,
		TotalConns: stats.TotalConns,
		IdleConns:  stats.IdleConns,
		StaleConns: stats.StaleConns,
	}
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck() error {
	if m.client == nil {
		return ErrClientNotInitialized
	}
	return m.client.Ping(nil)
}

// 错误定义
var (
	ErrClientNotInitialized = &CacheError{Message: "redis client not initialized"}
)

// CacheError 缓存错误
type CacheError struct {
	Message string
}

func (e *CacheError) Error() string {
	return e.Message
}
