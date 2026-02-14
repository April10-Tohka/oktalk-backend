// Package redis 定义 Redis Key 规则
// 命名规范: oktalk:{module}:{type}:{identifier}:{suffix}
package redis

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Key 前缀（遵循 oktalk:{module}:{type} 规范）
const (
	// 基础前缀
	KeyPrefix = "oktalk:"

	// 评测相关
	PrefixEvalResult = "oktalk:eval:result:"    // 评测完整结果 (Hash)
	PrefixEvalStatus = "oktalk:eval:status:"    // 评测状态

	// 示范音频
	PrefixDemoWord     = "oktalk:demo:audio:word:"     // 单词示范音频URL
	PrefixDemoSentence = "oktalk:demo:audio:sentence:" // 句子示范音频URL

	// 用户相关
	PrefixUserQuota   = "oktalk:user:quota:"   // 用户每日配额
	PrefixUserProfile = "oktalk:user:profile:" // 用户信息
	PrefixUserStats   = "oktalk:user:stats:"   // 用户统计
	PrefixUserToken   = "oktalk:user:token:"   // 用户Token

	// 临时数据
	PrefixTempUpload = "oktalk:temp:upload:" // 临时上传令牌

	// 反馈相关
	PrefixFeedbackText  = "oktalk:feedback:text:"  // LLM反馈文本缓存
	PrefixFeedbackAudio = "oktalk:feedback:audio:" // 反馈音频URL

	// 会话相关
	PrefixSession = "oktalk:session:" // 会话数据

	// 锁相关
	PrefixLock = "oktalk:lock:" // 分布式锁

	// 限流相关
	PrefixRateLimit = "oktalk:rate:" // 限流
)

// TTL 常量
const (
	TTLEvaluationResult = 7 * 24 * time.Hour  // 评测结果: 7天
	TTLDemoAudio        = 30 * 24 * time.Hour // 示范音频URL: 30天
	TTLUploadToken      = 5 * time.Minute     // 上传令牌: 5分钟
	TTLFeedbackText     = 7 * 24 * time.Hour  // LLM文本缓存: 7天
	TTLUserProfile      = 1 * time.Hour       // 用户信息: 1小时
	TTLUserStats        = 5 * time.Minute     // 用户统计: 5分钟
	TTLSession          = 24 * time.Hour      // 会话: 24小时
)

// NormalizeText 文本标准化（用于缓存key）
// 转小写、移除标点、空格替换为下划线
func NormalizeText(text string) string {
	// 转小写
	text = strings.ToLower(text)
	// 移除标点
	text = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(text, "")
	// 空格替换为下划线
	text = strings.ReplaceAll(text, " ", "_")
	// 限制长度
	if len(text) > 100 {
		text = text[:100]
	}
	return text
}

// ==================== 评测相关 Key ====================

// EvaluationKeys 评测相关 Key 构建器
type EvaluationKeys struct{}

// Result 评测完整结果缓存 Key (Hash)
// oktalk:eval:result:{evaluation_id}
func (EvaluationKeys) Result(evaluationID string) string {
	return PrefixEvalResult + evaluationID
}

// Status 评测状态 Key
// oktalk:eval:status:{evaluation_id}
func (EvaluationKeys) Status(evaluationID string) string {
	return PrefixEvalStatus + evaluationID
}

// ==================== 示范音频 Key ====================

// DemoAudioKeys 示范音频相关 Key 构建器
type DemoAudioKeys struct{}

// Word 单词示范音频URL Key
// oktalk:demo:audio:word:{normalized_word}
func (DemoAudioKeys) Word(word string) string {
	return PrefixDemoWord + NormalizeText(word)
}

// Sentence 句子示范音频URL Key
// oktalk:demo:audio:sentence:{normalized_sentence}
func (DemoAudioKeys) Sentence(sentence string) string {
	return PrefixDemoSentence + NormalizeText(sentence)
}

// ==================== 用户相关 Key ====================

// UserKeys 用户相关 Key 构建器
type UserKeys struct{}

// Quota 用户每日配额 Key
// oktalk:user:quota:{user_id}:{date}
func (UserKeys) Quota(userID string, date string) string {
	return fmt.Sprintf("%s%s:%s", PrefixUserQuota, userID, date)
}

// QuotaToday 用户今日配额 Key
func (UserKeys) QuotaToday(userID string) string {
	today := time.Now().Format("20060102")
	return fmt.Sprintf("%s%s:%s", PrefixUserQuota, userID, today)
}

// Profile 用户信息缓存 Key
// oktalk:user:profile:{user_id}
func (UserKeys) Profile(userID string) string {
	return PrefixUserProfile + userID
}

// Stats 用户统计缓存 Key
// oktalk:user:stats:{user_id}
func (UserKeys) Stats(userID string) string {
	return PrefixUserStats + userID
}

// Token 用户 Token 缓存 Key
// oktalk:user:token:{user_id}
func (UserKeys) Token(userID string) string {
	return PrefixUserToken + userID
}

// ==================== 临时数据 Key ====================

// TempKeys 临时数据相关 Key 构建器
type TempKeys struct{}

// UploadToken 临时上传令牌 Key
// oktalk:temp:upload:{token}
func (TempKeys) UploadToken(token string) string {
	return PrefixTempUpload + token
}

// ==================== 反馈相关 Key ====================

// FeedbackKeys 反馈相关 Key 构建器
type FeedbackKeys struct{}

// Text 反馈文本缓存 Key（基于分数、问题词、级别）
// oktalk:feedback:text:{score}:{problem_word}:{level}
func (FeedbackKeys) Text(score int, problemWord string, level string) string {
	return fmt.Sprintf("%s%d:%s:%s", PrefixFeedbackText, score, NormalizeText(problemWord), level)
}

// TextByEvaluation 基于评测ID的反馈文本 Key
// oktalk:feedback:text:{evaluation_id}
func (FeedbackKeys) TextByEvaluation(evaluationID string) string {
	return PrefixFeedbackText + evaluationID
}

// Audio 反馈音频URL Key
// oktalk:feedback:audio:{evaluation_id}
func (FeedbackKeys) Audio(evaluationID string) string {
	return PrefixFeedbackAudio + evaluationID
}

// ==================== 会话相关 Key ====================

// SessionKeys 会话相关 Key 构建器
type SessionKeys struct{}

// Data 会话数据缓存 Key
// oktalk:session:{session_id}
func (SessionKeys) Data(sessionID string) string {
	return PrefixSession + sessionID
}

// ==================== 锁相关 Key ====================

// LockKeys 分布式锁 Key 构建器
type LockKeys struct{}

// Evaluation 评测锁 Key
// oktalk:lock:eval:{evaluation_id}
func (LockKeys) Evaluation(evaluationID string) string {
	return PrefixLock + "eval:" + evaluationID
}

// User 用户锁 Key
// oktalk:lock:user:{user_id}
func (LockKeys) User(userID string) string {
	return PrefixLock + "user:" + userID
}

// ==================== 限流相关 Key ====================

// RateLimitKeys 限流 Key 构建器
type RateLimitKeys struct{}

// API API 限流 Key
// oktalk:rate:{api}:{user_id}
func (RateLimitKeys) API(api, userID string) string {
	return fmt.Sprintf("%s%s:%s", PrefixRateLimit, api, userID)
}

// ==================== 全局 Keys 构建器 ====================

// Keys 所有 Key 构建器
var Keys = struct {
	Evaluation EvaluationKeys
	DemoAudio  DemoAudioKeys
	User       UserKeys
	Temp       TempKeys
	Feedback   FeedbackKeys
	Session    SessionKeys
	Lock       LockKeys
	RateLimit  RateLimitKeys
}{}

// CalculateTodayRemainingTTL 计算当天剩余时间（用于每日配额）
func CalculateTodayRemainingTTL() time.Duration {
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	return endOfDay.Sub(now)
}
