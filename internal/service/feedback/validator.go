// Package feedback 提供反馈质量校验
package feedback

import (
	"pronunciation-correction-system/internal/model"
)

// Validator 反馈校验器
type Validator struct {
	minLength int
	maxLength int
}

// NewValidator 创建反馈校验器
func NewValidator() *Validator {
	return &Validator{
		minLength: 10,
		maxLength: 1000,
	}
}

// Validate 校验反馈质量
func (v *Validator) Validate(feedback *model.Feedback) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  []string{},
	}

	// 检查反馈文本长度
	if len(feedback.Text) < v.minLength {
		result.IsValid = false
		result.Errors = append(result.Errors, "反馈文本过短")
	}

	if len(feedback.Text) > v.maxLength {
		result.IsValid = false
		result.Errors = append(result.Errors, "反馈文本过长")
	}

	// TODO: 添加更多校验规则
	// - 检查是否包含不当内容
	// - 检查是否与评测结果相关
	// - 检查语法和格式

	return result
}

// ValidateAIResponse 校验 AI 生成的反馈
func (v *Validator) ValidateAIResponse(response string) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  []string{},
	}

	if response == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "AI 响应为空")
	}

	// TODO: 添加 AI 响应的特殊校验

	return result
}

// ValidationResult 校验结果
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
}
