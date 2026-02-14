// Package cache 提供评测相关缓存
// 评测结果使用 Hash 结构存储，支持部分字段更新
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"pronunciation-correction-system/internal/cache/redis"
)

// EvaluationCache 评测缓存
type EvaluationCache struct {
	commands *redis.Commands
}

// NewEvaluationCache 创建评测缓存
func NewEvaluationCache(commands *redis.Commands) *EvaluationCache {
	return &EvaluationCache{
		commands: commands,
	}
}

// EvaluationResult 评测结果缓存结构
// 对应 Hash 结构存储
type EvaluationResult struct {
	UserID           string `json:"user_id"`
	TargetText       string `json:"target_text"`
	RecognizedText   string `json:"recognized_text"`
	OverallScore     int    `json:"overall_score"`
	AccuracyScore    int    `json:"accuracy_score"`
	FluencyScore     int    `json:"fluency_score"`
	IntegrityScore   int    `json:"integrity_score"`
	FeedbackLevel    string `json:"feedback_level"`
	FeedbackText     string `json:"feedback_text"`
	FeedbackAudioURL string `json:"feedback_audio_url"`
	DemoAudioURL     string `json:"demo_audio_url"`
	DemoType         string `json:"demo_type"` // word/sentence
	Status           string `json:"status"`    // pending/processing/completed/failed
	CreatedAt        string `json:"created_at"`
	ErrorMessage     string `json:"error_message"`
}

// 评测状态常量
const (
	EvalStatusPending    = "pending"
	EvalStatusProcessing = "processing"
	EvalStatusCompleted  = "completed"
	EvalStatusFailed     = "failed"
)

// SetResult 设置完整评测结果缓存 (Hash)
func (c *EvaluationCache) SetResult(ctx context.Context, evaluationID string, result *EvaluationResult) error {
	key := redis.Keys.Evaluation.Result(evaluationID)

	// 设置 Hash 字段
	err := c.commands.HMSet(ctx, key,
		"user_id", result.UserID,
		"target_text", result.TargetText,
		"recognized_text", result.RecognizedText,
		"overall_score", strconv.Itoa(result.OverallScore),
		"accuracy_score", strconv.Itoa(result.AccuracyScore),
		"fluency_score", strconv.Itoa(result.FluencyScore),
		"integrity_score", strconv.Itoa(result.IntegrityScore),
		"feedback_level", result.FeedbackLevel,
		"feedback_text", result.FeedbackText,
		"feedback_audio_url", result.FeedbackAudioURL,
		"demo_audio_url", result.DemoAudioURL,
		"demo_type", result.DemoType,
		"status", result.Status,
		"created_at", result.CreatedAt,
		"error_message", result.ErrorMessage,
	)
	if err != nil {
		return err
	}

	// 设置过期时间 7天
	return c.commands.Expire(ctx, key, redis.TTLEvaluationResult)
}

// GetResult 获取完整评测结果缓存
func (c *EvaluationCache) GetResult(ctx context.Context, evaluationID string) (*EvaluationResult, error) {
	key := redis.Keys.Evaluation.Result(evaluationID)

	data, err := c.commands.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	// 检查是否为空
	if len(data) == 0 {
		return nil, nil
	}

	// 解析数据
	result := &EvaluationResult{
		UserID:           data["user_id"],
		TargetText:       data["target_text"],
		RecognizedText:   data["recognized_text"],
		FeedbackLevel:    data["feedback_level"],
		FeedbackText:     data["feedback_text"],
		FeedbackAudioURL: data["feedback_audio_url"],
		DemoAudioURL:     data["demo_audio_url"],
		DemoType:         data["demo_type"],
		Status:           data["status"],
		CreatedAt:        data["created_at"],
		ErrorMessage:     data["error_message"],
	}

	// 解析数字字段
	if v, ok := data["overall_score"]; ok {
		result.OverallScore, _ = strconv.Atoi(v)
	}
	if v, ok := data["accuracy_score"]; ok {
		result.AccuracyScore, _ = strconv.Atoi(v)
	}
	if v, ok := data["fluency_score"]; ok {
		result.FluencyScore, _ = strconv.Atoi(v)
	}
	if v, ok := data["integrity_score"]; ok {
		result.IntegrityScore, _ = strconv.Atoi(v)
	}

	return result, nil
}

// SetScores 设置评测分数（部分更新）
func (c *EvaluationCache) SetScores(ctx context.Context, evaluationID string, overall, accuracy, fluency, integrity int) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HMSet(ctx, key,
		"overall_score", strconv.Itoa(overall),
		"accuracy_score", strconv.Itoa(accuracy),
		"fluency_score", strconv.Itoa(fluency),
		"integrity_score", strconv.Itoa(integrity),
	)
}

// SetFeedback 设置反馈信息（部分更新）
func (c *EvaluationCache) SetFeedback(ctx context.Context, evaluationID, level, text, audioURL string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HMSet(ctx, key,
		"feedback_level", level,
		"feedback_text", text,
		"feedback_audio_url", audioURL,
	)
}

// SetDemoAudio 设置示范音频信息（部分更新）
func (c *EvaluationCache) SetDemoAudio(ctx context.Context, evaluationID, demoType, audioURL string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HMSet(ctx, key,
		"demo_type", demoType,
		"demo_audio_url", audioURL,
	)
}

// SetStatus 设置评测状态（部分更新）
func (c *EvaluationCache) SetStatus(ctx context.Context, evaluationID, status string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HSet(ctx, key, "status", status)
}

// SetError 设置错误信息（部分更新）
func (c *EvaluationCache) SetError(ctx context.Context, evaluationID, errorMessage string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HMSet(ctx, key,
		"status", EvalStatusFailed,
		"error_message", errorMessage,
	)
}

// GetStatus 获取评测状态
func (c *EvaluationCache) GetStatus(ctx context.Context, evaluationID string) (string, error) {
	key := redis.Keys.Evaluation.Result(evaluationID)
	status, err := c.commands.HGet(ctx, key, "status")
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return status, nil
}

// GetScores 获取评测分数
func (c *EvaluationCache) GetScores(ctx context.Context, evaluationID string) (overall, accuracy, fluency, integrity int, err error) {
	key := redis.Keys.Evaluation.Result(evaluationID)
	values, err := c.commands.HMGet(ctx, key, "overall_score", "accuracy_score", "fluency_score", "integrity_score")
	if err != nil {
		return 0, 0, 0, 0, err
	}

	if len(values) >= 4 {
		if v, ok := values[0].(string); ok {
			overall, _ = strconv.Atoi(v)
		}
		if v, ok := values[1].(string); ok {
			accuracy, _ = strconv.Atoi(v)
		}
		if v, ok := values[2].(string); ok {
			fluency, _ = strconv.Atoi(v)
		}
		if v, ok := values[3].(string); ok {
			integrity, _ = strconv.Atoi(v)
		}
	}

	return overall, accuracy, fluency, integrity, nil
}

// DeleteResult 删除评测结果缓存
func (c *EvaluationCache) DeleteResult(ctx context.Context, evaluationID string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.Del(ctx, key)
}

// ExistsResult 检查评测结果缓存是否存在
func (c *EvaluationCache) ExistsResult(ctx context.Context, evaluationID string) (bool, error) {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.Exists(ctx, key)
}

// InitProcessingResult 初始化处理中的评测结果
// 用于同步返回基础评分后立即缓存状态
func (c *EvaluationCache) InitProcessingResult(ctx context.Context, evaluationID, userID, targetText string, overall, accuracy, fluency, integrity int) error {
	result := &EvaluationResult{
		UserID:         userID,
		TargetText:     targetText,
		OverallScore:   overall,
		AccuracyScore:  accuracy,
		FluencyScore:   fluency,
		IntegrityScore: integrity,
		Status:         EvalStatusProcessing,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// 计算反馈级别
	if overall >= 90 {
		result.FeedbackLevel = "S"
	} else if overall >= 70 {
		result.FeedbackLevel = "A"
	} else if overall >= 50 {
		result.FeedbackLevel = "B"
	} else {
		result.FeedbackLevel = "C"
	}

	return c.SetResult(ctx, evaluationID, result)
}

// CompleteResult 完成评测结果
// 更新状态为 completed，设置所有反馈信息
func (c *EvaluationCache) CompleteResult(ctx context.Context, evaluationID, feedbackText, feedbackAudioURL, demoType, demoAudioURL string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.HMSet(ctx, key,
		"status", EvalStatusCompleted,
		"feedback_text", feedbackText,
		"feedback_audio_url", feedbackAudioURL,
		"demo_type", demoType,
		"demo_audio_url", demoAudioURL,
	)
}

// GetResultField 获取单个字段值
func (c *EvaluationCache) GetResultField(ctx context.Context, evaluationID, field string) (string, error) {
	key := redis.Keys.Evaluation.Result(evaluationID)
	value, err := c.commands.HGet(ctx, key, field)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// GetTTL 获取评测结果缓存剩余时间
func (c *EvaluationCache) GetTTL(ctx context.Context, evaluationID string) (time.Duration, error) {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.TTL(ctx, key)
}

// RefreshTTL 刷新评测结果缓存过期时间
func (c *EvaluationCache) RefreshTTL(ctx context.Context, evaluationID string) error {
	key := redis.Keys.Evaluation.Result(evaluationID)
	return c.commands.Expire(ctx, key, redis.TTLEvaluationResult)
}

// BatchGetStatus 批量获取评测状态
func (c *EvaluationCache) BatchGetStatus(ctx context.Context, evaluationIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, id := range evaluationIDs {
		status, err := c.GetStatus(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get status for %s: %w", id, err)
		}
		if status != "" {
			result[id] = status
		}
	}
	return result, nil
}
