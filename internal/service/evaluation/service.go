// Package evaluation 提供发音评测业务逻辑
// 负责协调语音评测、评分、分析等功能
package evaluation

import (
	"context"
	"fmt"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
)

// ServiceImpl 评测服务实现
type ServiceImpl struct {
	evaluationProvider domain.EvaluationProvider // 语音评测（讯飞）
	scorer             *Scorer                   // 评分器
	analyzer           *Analyzer                 // 分析器
}

// NewService 创建评测服务
func NewService(evaluationProvider domain.EvaluationProvider) Service {
	return &ServiceImpl{
		evaluationProvider: evaluationProvider,
		scorer:             NewScorer(),
		analyzer:           NewAnalyzer(),
	}
}

// Submit 提交发音评测
func (s *ServiceImpl) Submit(ctx context.Context, req *EvaluationRequest) (*model.Evaluation, error) {
	// 1. 调用讯飞语音评测 API（通过 domain.EvaluationProvider 接口）
	assessResult, err := s.evaluationProvider.Assess(ctx, req.Text, req.AudioData)
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	// 2. 将领域评测结果转换为内部原始结果进行评分
	rawResult := convertToRaw(assessResult)

	// 3. 计算综合评分
	overallScore := s.scorer.CalculateOverallScore(rawResult)
	level := s.scorer.GetLevel(overallScore)

	// 4. 分析发音问题
	_ = s.analyzer.AnalyzeProblems(rawResult)

	// 5. 构建评测记录（float64 → int 转换）
	evaluation := &model.Evaluation{
		UserID:         req.UserID,
		TargetText:     req.Text,
		OverallScore:   int(overallScore),
		AccuracyScore:  int(assessResult.Accuracy),
		FluencyScore:   int(assessResult.Fluency),
		IntegrityScore: int(assessResult.Completeness),
		FeedbackLevel:  level,
		Status:         "processing",
	}

	// TODO: 保存到数据库
	// TODO: 触发异步反馈生成

	return evaluation, nil
}

// GetResult 获取评测结果
func (s *ServiceImpl) GetResult(ctx context.Context, evaluationID string) (*model.Evaluation, error) {
	// TODO: 从缓存获取 → 缓存未命中则从数据库获取
	_ = ctx
	_ = evaluationID
	return nil, nil
}

// GetHistory 获取评测历史
func (s *ServiceImpl) GetHistory(ctx context.Context, userID string, page, pageSize int) ([]*model.Evaluation, int64, error) {
	// TODO: 实现获取评测历史
	_ = ctx
	_ = userID
	_ = page
	_ = pageSize
	return nil, 0, nil
}

// EvaluationRequest 评测请求
type EvaluationRequest struct {
	UserID    string `json:"user_id"`
	TextID    string `json:"text_id"`
	Text      string `json:"text"`
	AudioData []byte `json:"audio_data"`
	AudioURL  string `json:"audio_url"`
}

// convertToRaw 将领域评测结果转换为内部原始结果
func convertToRaw(result *domain.EvaluationResult) *RawEvaluationResult {
	raw := &RawEvaluationResult{
		Accuracy:     result.Accuracy,
		Fluency:      result.Fluency,
		Completeness: result.Completeness,
		Intonation:   result.Intonation,
		Words:        make([]WordResult, len(result.Words)),
	}
	for i, w := range result.Words {
		word := WordResult{
			Word:      w.Word,
			Score:     w.Score,
			StartTime: w.BeginTime,
			EndTime:   w.EndTime,
		}
		raw.Words[i] = word
	}
	return raw
}
