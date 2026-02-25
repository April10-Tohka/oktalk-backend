package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// ===================== Chat 缓存结构 =====================

// ChatResult 对话结果
type ChatResult struct {
	TaskID         string `json:"task_id"`
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	UserText       string `json:"user_text"`
	UserAudioURL   string `json:"user_audio_url,omitempty"`
	DurationMs     int    `json:"duration_ms"`
	AIReply        string `json:"ai_reply"`
	AudioURL       string `json:"audio_url,omitempty"`
	AIDurationMs   int    `json:"ai_duration_ms,omitempty"`
	CreatedAt      int64  `json:"created_at"`
}

// SessionMessage 会话消息
type SessionMessage struct {
	Role      string `json:"role"` // user / assistant
	Content   string `json:"content"`
	AudioURL  string `json:"audio_url,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// PagedList 分页列表（通用）
type PagedList struct {
	Items      json.RawMessage `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	Total      int64           `json:"total"`
	TotalPages int             `json:"total_pages"`
}

// ===================== ChatCache =====================

// ChatCache 对话缓存操作
type ChatCache struct {
	rdb *redis.Client
}

// NewChatCache 创建 ChatCache
func NewChatCache(rdb *redis.Client) *ChatCache {
	return &ChatCache{rdb: rdb}
}

// SetChatResult 写入对话结果
func (c *ChatCache) SetChatResult(ctx context.Context, conversationID string, result *ChatResult) error {
	key := fmt.Sprintf(KeyChatResult, conversationID)
	return SetJSON(ctx, c.rdb, key, result, TTLChatResult)
}

// GetChatResult 获取对话结果
func (c *ChatCache) GetChatResult(ctx context.Context, conversationID string) (*ChatResult, bool, error) {
	key := fmt.Sprintf(KeyChatResult, conversationID)
	return GetJSON[ChatResult](ctx, c.rdb, key)
}

// SetChatHistory 写入对话历史分页
func (c *ChatCache) SetChatHistory(ctx context.Context, userID string, page int, data *PagedList) error {
	key := fmt.Sprintf(KeyChatHistory, userID, page)
	return SetJSON(ctx, c.rdb, key, data, TTLHistoryList)
}

// GetChatHistory 获取对话历史分页
func (c *ChatCache) GetChatHistory(ctx context.Context, userID string, page int) (*PagedList, bool, error) {
	key := fmt.Sprintf(KeyChatHistory, userID, page)
	return GetJSON[PagedList](ctx, c.rdb, key)
}

// InvalidateChatHistory 删除该用户所有页的历史缓存
// 使用 SCAN 匹配 chat:history:{user_id}:* 后批量删除
func (c *ChatCache) InvalidateChatHistory(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("chat:history:%s:*", userID)
	return deleteByPattern(ctx, c.rdb, pattern)
}

// ===================== Session 方法 =====================

// AppendSessionMessage 追加会话消息到 List
func (c *ChatCache) AppendSessionMessage(ctx context.Context, conversationID string, msg *SessionMessage) error {
	key := fmt.Sprintf(KeyChatSession, conversationID)
	return RPush(ctx, c.rdb, key, msg, TTLChatSession)
}

// GetSessionMessages 获取会话所有消息
func (c *ChatCache) GetSessionMessages(ctx context.Context, conversationID string) ([]*SessionMessage, error) {
	key := fmt.Sprintf(KeyChatSession, conversationID)
	data, err := c.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]*SessionMessage, 0, len(data))
	for _, item := range data {
		var msg SessionMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			slog.Warn("parse session message failed", "error", err)
			continue
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}
