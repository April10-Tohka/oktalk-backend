// Package validator 提供请求参数验证
package validator

import (
	"regexp"
	"strings"
)

// Validator 验证器
type Validator struct{}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateEmail 验证邮箱
func (v *Validator) ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidatePassword 验证密码
func (v *Validator) ValidatePassword(password string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if len(password) < 6 {
		result.IsValid = false
		result.Errors = append(result.Errors, "password must be at least 6 characters")
	}

	if len(password) > 50 {
		result.IsValid = false
		result.Errors = append(result.Errors, "password must be at most 50 characters")
	}

	return result
}

// ValidateUsername 验证用户名
func (v *Validator) ValidateUsername(username string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if len(username) < 3 {
		result.IsValid = false
		result.Errors = append(result.Errors, "username must be at least 3 characters")
	}

	if len(username) > 30 {
		result.IsValid = false
		result.Errors = append(result.Errors, "username must be at most 30 characters")
	}

	pattern := `^[a-zA-Z0-9_]+$`
	if matched, _ := regexp.MatchString(pattern, username); !matched {
		result.IsValid = false
		result.Errors = append(result.Errors, "username can only contain letters, numbers and underscores")
	}

	return result
}

// ValidateText 验证文本
func (v *Validator) ValidateText(text string, minLen, maxLen int) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	text = strings.TrimSpace(text)

	if len(text) < minLen {
		result.IsValid = false
		result.Errors = append(result.Errors, "text is too short")
	}

	if len(text) > maxLen {
		result.IsValid = false
		result.Errors = append(result.Errors, "text is too long")
	}

	return result
}

// ValidateUUID 验证 UUID
func (v *Validator) ValidateUUID(id string) bool {
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(pattern, strings.ToLower(id))
	return matched
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
}

// Error 返回第一个错误
func (r *ValidationResult) Error() string {
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	return ""
}

// AllErrors 返回所有错误
func (r *ValidationResult) AllErrors() string {
	return strings.Join(r.Errors, "; ")
}
