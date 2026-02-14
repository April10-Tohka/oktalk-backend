// Package feedback 提供反馈生成服务
// 负责根据评测结果生成个性化反馈
package feedback

import (
	"context"
	"fmt"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"
)

// Generator 反馈生成器
type Generator struct {
	llm    domain.LLMProvider // LLM 服务（通义千问等）
	levels *Levels
}

// NewGenerator 创建反馈生成器
func NewGenerator(llm domain.LLMProvider) *Generator {
	return &Generator{
		llm:    llm,
		levels: NewLevels(),
	}
}

// Generate 生成反馈
func (g *Generator) Generate(ctx context.Context, evaluation *model.Evaluation) (*model.Feedback, error) {
	// 1. 确定反馈等级
	levelRule := g.levels.GetLevel(float64(evaluation.OverallScore))

	// 2. 构建 Prompt
	prompt := g.BuildPrompt(evaluation)

	// 3. 调用 LLM 生成个性化反馈文本（通过 domain.LLMProvider 接口）
	systemPrompt := "你是一位专业的英语发音教练。请用简洁友好的语气提供反馈。"
	feedbackText, err := g.llm.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		// LLM 失败时使用模板降级
		feedbackText = g.levels.GetFeedbackTemplate(levelRule.Level)
	}

	// 4. 构建反馈结构
	feedback := &model.Feedback{
		EvaluationID: evaluation.ID,
		Text:         feedbackText,
		Level:        levelRule.Level,
	}

	return feedback, nil
}

// GenerateWithAI 使用 AI 生成反馈文本
func (g *Generator) GenerateWithAI(ctx context.Context, prompt string) (string, error) {
	systemPrompt := "你是一位专业的英语发音教练。请用简洁友好的语气提供反馈和改进建议。"
	return g.llm.Chat(ctx, systemPrompt, prompt)
}

// BuildPrompt 构建生成反馈的 Prompt
func (g *Generator) BuildPrompt(evaluation *model.Evaluation) string {
	return fmt.Sprintf(`评测文本: %s
综合评分: %d
准确度: %d
流利度: %d
完整度: %d

请提供：
1. 简短的总体评价（1-2句话）
2. 具体的发音问题分析
3. 针对性的改进建议
4. 鼓励性的结语`,
		evaluation.TargetText,
		evaluation.OverallScore,
		evaluation.AccuracyScore,
		evaluation.FluencyScore,
		evaluation.IntegrityScore,
	)
}
