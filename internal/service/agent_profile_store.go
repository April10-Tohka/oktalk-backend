package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ===================== 长期语义记忆存储（Agent 画像） =====================
//
// 这里存储的是"教练人格层"的长期记忆：孩子的名字、年龄段、兴趣话题——
// 用于让"龙宝 OK"跨会话记住每个小朋友，个性化语气与难度（对应设计文档 P2 长期语义记忆）。
//
// 设计选择：复用 Redis 做持久化（项目已具备 RedisClient，且 initRedis 失败会自动降级）。
// 不写入 model.UserProfile（那张表由 UserService/AuthService 管理，字段是 Age/Gender/Bio/统计），
// 避免扰动已校验的共享模型与 DB 迁移；Agent 画像以 JSON 独立存储，彼此解耦、风险最低。
// 若 Redis 不可用（client 为 nil），自动退化为进程内内存存储，保证会话内行为不中断。

// AgentProfileStore 长期语义记忆存储接口
type AgentProfileStore interface {
	// Load 按 userID 读取 Agent 画像；未找到返回 (nil, nil)。
	Load(ctx context.Context, userID string) (*UserProfile, error)
	// Save 覆盖写入指定 userID 的 Agent 画像。
	Save(ctx context.Context, userID string, p *UserProfile) error
}

// redisKeyForAgentProfile 生成 Redis key
func redisKeyForAgentProfile(userID string) string {
	return fmt.Sprintf("oktalk:agent:profile:%s", userID)
}

// redisAgentProfileStore Redis 实现的 Agent 画像存储
type redisAgentProfileStore struct {
	client *redis.Client
}

func (r *redisAgentProfileStore) Load(ctx context.Context, userID string) (*UserProfile, error) {
	data, err := r.client.Get(ctx, redisKeyForAgentProfile(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p UserProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *redisAgentProfileStore) Save(ctx context.Context, userID string, p *UserProfile) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	// 7 天过期，避免僵尸数据常驻；会话活跃时会不断刷新 TTL。
	return r.client.Set(ctx, redisKeyForAgentProfile(userID), data, 7*24*time.Hour).Err()
}

// memoryAgentProfileStore 进程内兜底存储（Redis 不可用时的降级方案）
type memoryAgentProfileStore struct {
	mu sync.Mutex
	m  map[string]*UserProfile
}

func (s *memoryAgentProfileStore) Load(_ context.Context, userID string) (*UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.m[userID]
	if !ok {
		return nil, nil
	}
	// 返回副本，避免调用方修改污染内部状态
	cp := *p
	return &cp, nil
}

func (s *memoryAgentProfileStore) Save(_ context.Context, userID string, p *UserProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.m[userID] = &cp
	return nil
}

// NewAgentProfileStore 创建 Agent 画像存储。
// client 为 nil（Redis 未初始化）时返回进程内内存存储降级实现。
func NewAgentProfileStore(client *redis.Client) AgentProfileStore {
	if client == nil {
		return &memoryAgentProfileStore{m: make(map[string]*UserProfile)}
	}
	return &redisAgentProfileStore{client: client}
}
