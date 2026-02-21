package xf

import (
	"context"
	"fmt"
	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"
)

// XFEvaluationAdapter 科大讯飞语音评测适配器
// 实现 domain.EvaluationProvider 接口
type XFEvaluationAdapter struct {
	client *internalClient
}

// 编译时检查：确保 XFEvaluationAdapter 实现了 domain.EvaluationProvider 接口
var _ domain.EvaluationProvider = (*XFEvaluationAdapter)(nil)

// NewXFEvaluationAdapter 创建科大讯飞语音评测适配器
func NewXFEvaluationAdapter(cfg config.XunFeiConfig) *XFEvaluationAdapter {
	return &XFEvaluationAdapter{
		client: newInternalClient(cfg),
	}
}

// Assess 执行语音评测
func (a *XFEvaluationAdapter) Assess(ctx context.Context, text string, audioData []byte) (*domain.EvaluationResult, error) {
	logger.InfoContext(ctx, "xf evaluation: starting assess",
		"text_length", len(text),
		"audio_bytes", len(audioData))

	req := &speechAssessRequest{
		Text:      text,
		AudioData: audioData,
		Category:  "read_sentence",
		Language:  "en_vip",
	}

	result, err := a.client.speechAssess(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("xf speech assess failed: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("xf speech assess returned nil result")
	}

	logger.InfoContext(ctx, "xf evaluation: assess completed",
		"total_score", result.TotalScore,
		"accuracy", result.Accuracy,
		"fluency", result.Fluency,
		"word_count", len(result.Words))

	// 将内部 SDK 结果转换为领域层结果
	return convertToResult(result), nil
}

// Close 关闭客户端
func (a *XFEvaluationAdapter) Close() error {
	return a.client.close()
}

// convertToResult 将 SDK 内部结果转换为领域层 EvaluationResult
func convertToResult(sdkResult *speechAssessResult) *domain.EvaluationResult {
	result := &domain.EvaluationResult{
		TotalScore:   sdkResult.TotalScore,
		Accuracy:     sdkResult.Accuracy,
		Fluency:      sdkResult.Fluency,
		Completeness: sdkResult.Completeness,
		Intonation:   sdkResult.Intonation,
		Words:        make([]domain.WordEvaluationResult, len(sdkResult.Words)),
	}

	for i, w := range sdkResult.Words {
		word := domain.WordEvaluationResult{
			Word:      w.Word,
			Score:     w.Score,
			BeginTime: w.BeginTime,
			EndTime:   w.EndTime,
			Phonemes:  make([]domain.PhonemeEvaluationResult, len(w.Phonemes)),
		}
		for j, p := range w.Phonemes {
			word.Phonemes[j] = domain.PhonemeEvaluationResult{
				Phoneme:   p.Phoneme,
				Score:     p.Score,
				BeginTime: p.BeginTime,
				EndTime:   p.EndTime,
			}
		}
		result.Words[i] = word
	}

	return result
}
