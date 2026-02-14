// Package redis 提供 Redis 操作封装
package redis

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// Commands Redis 命令封装
type Commands struct {
	client *Client
}

// NewCommands 创建命令封装
func NewCommands(client *Client) *Commands {
	return &Commands{client: client}
}

// ==================== String 操作 ====================

// Get 获取字符串值
func (c *Commands) Get(ctx context.Context, key string) (string, error) {
	return c.client.rdb.Get(ctx, key).Result()
}

// GetBytes 获取字节值
func (c *Commands) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.client.rdb.Get(ctx, key).Bytes()
}

// Set 设置字符串值
func (c *Commands) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.rdb.Set(ctx, key, value, expiration).Err()
}

// SetNX 设置值（仅当 key 不存在时）
func (c *Commands) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.rdb.SetNX(ctx, key, value, expiration).Result()
}

// SetEX 设置值并指定过期时间
func (c *Commands) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.rdb.SetEX(ctx, key, value, expiration).Err()
}

// GetSet 获取旧值并设置新值
func (c *Commands) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	return c.client.rdb.GetSet(ctx, key, value).Result()
}

// MGet 批量获取
func (c *Commands) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.client.rdb.MGet(ctx, keys...).Result()
}

// MSet 批量设置
func (c *Commands) MSet(ctx context.Context, values ...interface{}) error {
	return c.client.rdb.MSet(ctx, values...).Err()
}

// ==================== Key 操作 ====================

// Del 删除 key
func (c *Commands) Del(ctx context.Context, keys ...string) error {
	return c.client.rdb.Del(ctx, keys...).Err()
}

// Exists 检查 key 是否存在
func (c *Commands) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.rdb.Exists(ctx, key).Result()
	return result > 0, err
}

// Expire 设置过期时间
func (c *Commands) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.rdb.Expire(ctx, key, expiration).Err()
}

// ExpireAt 设置过期时间点
func (c *Commands) ExpireAt(ctx context.Context, key string, tm time.Time) error {
	return c.client.rdb.ExpireAt(ctx, key, tm).Err()
}

// TTL 获取剩余过期时间
func (c *Commands) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.rdb.TTL(ctx, key).Result()
}

// Keys 按模式获取 key 列表
func (c *Commands) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.rdb.Keys(ctx, pattern).Result()
}

// Scan 扫描 key
func (c *Commands) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.client.rdb.Scan(ctx, cursor, match, count).Result()
}

// ==================== JSON 操作 ====================

// GetJSON 获取 JSON 对象
func (c *Commands) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// SetJSON 设置 JSON 对象
func (c *Commands) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.rdb.Set(ctx, key, data, expiration).Err()
}

// ==================== Hash 操作 ====================

// HGet 获取 Hash 字段值
func (c *Commands) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.rdb.HGet(ctx, key, field).Result()
}

// HSet 设置 Hash 字段值
func (c *Commands) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.rdb.HSet(ctx, key, values...).Err()
}

// HSetNX 设置 Hash 字段值（仅当字段不存在时）
func (c *Commands) HSetNX(ctx context.Context, key, field string, value interface{}) (bool, error) {
	return c.client.rdb.HSetNX(ctx, key, field, value).Result()
}

// HGetAll 获取所有 Hash 字段
func (c *Commands) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.rdb.HGetAll(ctx, key).Result()
}

// HMGet 批量获取 Hash 字段
func (c *Commands) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return c.client.rdb.HMGet(ctx, key, fields...).Result()
}

// HMSet 批量设置 Hash 字段
func (c *Commands) HMSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.rdb.HMSet(ctx, key, values...).Err()
}

// HDel 删除 Hash 字段
func (c *Commands) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.rdb.HDel(ctx, key, fields...).Err()
}

// HExists 检查 Hash 字段是否存在
func (c *Commands) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.client.rdb.HExists(ctx, key, field).Result()
}

// HLen 获取 Hash 字段数量
func (c *Commands) HLen(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.HLen(ctx, key).Result()
}

// HIncrBy Hash 字段增加整数
func (c *Commands) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.client.rdb.HIncrBy(ctx, key, field, incr).Result()
}

// HIncrByFloat Hash 字段增加浮点数
func (c *Commands) HIncrByFloat(ctx context.Context, key, field string, incr float64) (float64, error) {
	return c.client.rdb.HIncrByFloat(ctx, key, field, incr).Result()
}

// ==================== 计数操作 ====================

// Incr 自增
func (c *Commands) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.Incr(ctx, key).Result()
}

// IncrBy 增加指定值
func (c *Commands) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.rdb.IncrBy(ctx, key, value).Result()
}

// Decr 自减
func (c *Commands) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.Decr(ctx, key).Result()
}

// DecrBy 减少指定值
func (c *Commands) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.rdb.DecrBy(ctx, key, value).Result()
}

// IncrByFloat 浮点数增加
func (c *Commands) IncrByFloat(ctx context.Context, key string, value float64) (float64, error) {
	return c.client.rdb.IncrByFloat(ctx, key, value).Result()
}

// ==================== List 操作 ====================

// LPush 列表左侧插入
func (c *Commands) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.rdb.LPush(ctx, key, values...).Err()
}

// RPush 列表右侧插入
func (c *Commands) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.rdb.RPush(ctx, key, values...).Err()
}

// LPop 列表左侧弹出
func (c *Commands) LPop(ctx context.Context, key string) (string, error) {
	return c.client.rdb.LPop(ctx, key).Result()
}

// RPop 列表右侧弹出
func (c *Commands) RPop(ctx context.Context, key string) (string, error) {
	return c.client.rdb.RPop(ctx, key).Result()
}

// BLPop 阻塞左侧弹出
func (c *Commands) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return c.client.rdb.BLPop(ctx, timeout, keys...).Result()
}

// BRPop 阻塞右侧弹出
func (c *Commands) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return c.client.rdb.BRPop(ctx, timeout, keys...).Result()
}

// LRange 获取列表范围
func (c *Commands) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.rdb.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func (c *Commands) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.LLen(ctx, key).Result()
}

// LTrim 裁剪列表
func (c *Commands) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.client.rdb.LTrim(ctx, key, start, stop).Err()
}

// ==================== Set 操作 ====================

// SAdd 集合添加
func (c *Commands) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.rdb.SAdd(ctx, key, members...).Err()
}

// SRem 集合删除
func (c *Commands) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.rdb.SRem(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (c *Commands) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.rdb.SMembers(ctx, key).Result()
}

// SIsMember 检查是否是集合成员
func (c *Commands) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.rdb.SIsMember(ctx, key, member).Result()
}

// SCard 获取集合大小
func (c *Commands) SCard(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.SCard(ctx, key).Result()
}

// ==================== Sorted Set 操作 ====================

// ZAdd 有序集合添加
func (c *Commands) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	return c.client.rdb.ZAdd(ctx, key, members...).Err()
}

// ZRem 有序集合删除
func (c *Commands) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.rdb.ZRem(ctx, key, members...).Err()
}

// ZRange 获取有序集合范围
func (c *Commands) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.rdb.ZRange(ctx, key, start, stop).Result()
}

// ZRangeWithScores 获取有序集合范围（带分数）
func (c *Commands) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.client.rdb.ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZScore 获取成员分数
func (c *Commands) ZScore(ctx context.Context, key, member string) (float64, error) {
	return c.client.rdb.ZScore(ctx, key, member).Result()
}

// ZCard 获取有序集合大小
func (c *Commands) ZCard(ctx context.Context, key string) (int64, error) {
	return c.client.rdb.ZCard(ctx, key).Result()
}

// ==================== 分布式锁 ====================

// Lock 获取分布式锁
func (c *Commands) Lock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	return c.client.rdb.SetNX(ctx, key, value, expiration).Result()
}

// Unlock 释放分布式锁（仅当值匹配时）
func (c *Commands) Unlock(ctx context.Context, key, value string) error {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	return script.Run(ctx, c.client.rdb, []string{key}, value).Err()
}

// ==================== 辅助函数 ====================

// IsNil 检查是否为 redis.Nil 错误
func IsNil(err error) bool {
	return err == redis.Nil
}

// ParseInt 解析整数
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ParseInt64 解析64位整数
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseFloat 解析浮点数
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
