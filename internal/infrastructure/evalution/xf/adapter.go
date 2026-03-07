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
func (a *XFEvaluationAdapter) Assess(ctx context.Context, text string, audioData []byte, category string) (*domain.EvaluationResult, error) {
	logger.InfoContext(ctx, "xf evaluation: starting assess",
		"text_length", len(text),
		"audio_bytes", len(audioData),
		"category", category)

	req := &speechAssessRequest{
		Text:      text,
		AudioData: audioData,
		Category:  category,
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
		"sid", result.Sid,
		"total_score", result.TotalScore,
		"accuracy_score", result.AccuracyScore,
		"fluency_score", result.FluencyScore,
		"integrity_score", result.IntegrityScore,
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
		Sid:            sdkResult.Sid,
		TotalScore:     (sdkResult.TotalScore),
		AccuracyScore:  (sdkResult.AccuracyScore),
		FluencyScore:   (sdkResult.FluencyScore),
		IntegrityScore: (sdkResult.IntegrityScore),
		StandardScore:  (sdkResult.StandardScore),
		Words:          make([]domain.WordEvaluationResult, len(sdkResult.Words)),
		RawXML:         sdkResult.RawXML,
		IsRejected:     sdkResult.IsRejected,
		ExceptInfo:     sdkResult.ExceptInfo,
	}

	for i, w := range sdkResult.Words {
		word := domain.WordEvaluationResult{
			Word:      w.Word,
			Score:     w.Score,
			BeginTime: w.BeginTime,
			EndTime:   w.EndTime,
			DpMessage: w.DpMessage,
			// Phonemes:  make([]domain.PhonemeEvaluationResult, len(w.Phonemes)),
		}
		// TODO: 后续完善音素级结果转换
		// for j, p := range w.Phonemes {
		// 	word.Phonemes[j] = domain.PhonemeEvaluationResult{
		// 		Phoneme:   p.Phoneme,
		// 		Score:     p.Score,
		// 		BeginTime: p.BeginTime,
		// 		EndTime:   p.EndTime,
		// 	}
		// }
		result.Words[i] = word
	}

	return result
}
