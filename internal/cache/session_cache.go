// Package cache 提供会话相关缓存
// 用于存储用户登录会话信息
package cache

import (
	"context"
	"encoding/json"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// SessionCache 会话缓存
type SessionCache struct {
	commands *redis.Commands
}

// NewSessionCache 创建会话缓存
func NewSessionCache(commands *redis.Commands) *SessionCache {
	return &SessionCache{
		commands: commands,
	}
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	Token       string    `json:"token"`
	UserAgent   string    `json:"user_agent"`
	IP          string    `json:"ip"`
	DeviceType  string    `json:"device_type"`  // web/mobile/desktop
	LoginAt     time.Time `json:"login_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// SetSession 设置会话信息
func (c *SessionCache) SetSession(ctx context.Context, sessionID string, info *SessionInfo, ttl time.Duration) error {
	key := redis.Keys.Session.Data(sessionID)

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if ttl <= 0 {
		ttl = redis.TTLSession
	}

	return c.commands.Set(ctx, key, data, ttl)
}

// GetSession 获取会话信息
func (c *SessionCache) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	key := redis.Keys.Session.Data(sessionID)

	data, err := c.commands.GetBytes(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var info SessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// DeleteSession 删除会话
func (c *SessionCache) DeleteSession(ctx context.Context, sessionID string) error {
	key := redis.Keys.Session.Data(sessionID)
	return c.commands.Del(ctx, key)
}

// ExistsSession 检查会话是否存在
func (c *SessionCache) ExistsSession(ctx context.Context, sessionID string) (bool, error) {
	key := redis.Keys.Session.Data(sessionID)
	return c.commands.Exists(ctx, key)
}

// RefreshSession 刷新会话过期时间
func (c *SessionCache) RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := redis.Keys.Session.Data(sessionID)

	if ttl <= 0 {
		ttl = redis.TTLSession
	}

	return c.commands.Expire(ctx, key, ttl)
}

// UpdateLastActive 更新会话最后活动时间
func (c *SessionCache) UpdateLastActive(ctx context.Context, sessionID string) error {
	// 获取当前会话信息
	info, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if info == nil {
		return nil // 会话不存在
	}

	// 更新最后活动时间
	info.LastActiveAt = time.Now()

	// 计算剩余 TTL
	ttl, err := c.GetSessionTTL(ctx, sessionID)
	if err != nil {
		ttl = redis.TTLSession
	}

	return c.SetSession(ctx, sessionID, info, ttl)
}

// GetSessionTTL 获取会话剩余时间
func (c *SessionCache) GetSessionTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	key := redis.Keys.Session.Data(sessionID)
	return c.commands.TTL(ctx, key)
}

// ValidateSession 验证会话
// 检查会话是否存在、是否过期、token是否匹配
func (c *SessionCache) ValidateSession(ctx context.Context, sessionID, token string) (*SessionInfo, bool, error) {
	info, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return nil, false, err
	}
	if info == nil {
		return nil, false, nil // 会话不存在
	}

	// 检查 token
	if info.Token != token {
		return nil, false, nil // token 不匹配
	}

	// 检查过期时间
	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		// 已过期，删除会话
		_ = c.DeleteSession(ctx, sessionID)
		return nil, false, nil
	}

	return info, true, nil
}

// CreateSession 创建新会话
func (c *SessionCache) CreateSession(ctx context.Context, info *SessionInfo, ttl time.Duration) error {
	// 设置创建时间和过期时间
	now := time.Now()
	if info.LoginAt.IsZero() {
		info.LoginAt = now
	}
	if info.LastActiveAt.IsZero() {
		info.LastActiveAt = now
	}
	if info.ExpiresAt.IsZero() && ttl > 0 {
		info.ExpiresAt = now.Add(ttl)
	}

	return c.SetSession(ctx, info.SessionID, info, ttl)
}

// ExtendSession 延长会话有效期
func (c *SessionCache) ExtendSession(ctx context.Context, sessionID string, extension time.Duration) error {
	// 获取当前会话信息
	info, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if info == nil {
		return nil
	}

	// 更新过期时间
	info.ExpiresAt = time.Now().Add(extension)
	info.LastActiveAt = time.Now()

	return c.SetSession(ctx, sessionID, info, extension)
}

// GetUserSessions 获取用户所有会话ID列表（需要维护用户会话列表）
// 注意：这需要额外的数据结构来维护用户与会话的关系
func (c *SessionCache) SetUserSession(ctx context.Context, userID, sessionID string, ttl time.Duration) error {
	key := redis.PrefixUserToken + userID + ":sessions"
	// 添加到用户会话集合
	if err := c.commands.SAdd(ctx, key, sessionID); err != nil {
		return err
	}
	return c.commands.Expire(ctx, key, ttl)
}

// RemoveUserSession 从用户会话列表移除会话
func (c *SessionCache) RemoveUserSession(ctx context.Context, userID, sessionID string) error {
	key := redis.PrefixUserToken + userID + ":sessions"
	return c.commands.SRem(ctx, key, sessionID)
}

// GetUserSessionIDs 获取用户所有会话ID
func (c *SessionCache) GetUserSessionIDs(ctx context.Context, userID string) ([]string, error) {
	key := redis.PrefixUserToken + userID + ":sessions"
	return c.commands.SMembers(ctx, key)
}

// DeleteAllUserSessions 删除用户所有会话
func (c *SessionCache) DeleteAllUserSessions(ctx context.Context, userID string) error {
	// 获取所有会话ID
	sessionIDs, err := c.GetUserSessionIDs(ctx, userID)
	if err != nil {
		return err
	}

	// 删除每个会话
	for _, sessionID := range sessionIDs {
		if err := c.DeleteSession(ctx, sessionID); err != nil {
			continue // 继续删除其他会话
		}
	}

	// 删除用户会话列表
	key := redis.PrefixUserToken + userID + ":sessions"
	return c.commands.Del(ctx, key)
}
